//go:build !(js && wasm)

package gfx

import (
	"image"
	"image/color"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// redSquare is a drawing of a red square filling its own 10 by 10 space.
const redSquare = `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 10 10">` +
	`<rect width="10" height="10" fill="#ff0000"/></svg>`

// thinLine is a drawing of a line two units wide straight across its own
// 100 by 100 space, drawn as a stroke rather than as a filled shape, so it
// covers a fiftieth of the drawing at whatever size it is rendered.
const thinLine = `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">` +
	`<path d="M0 50H100" fill="none" stroke="#000000" stroke-width="2"/></svg>`

// waitRaster blits nothing but asks the sprite for a raster at scale until
// the one it started has been drawn, which happens on another goroutine.
func waitRaster(t *testing.T, s *Sprite, scale float64) *ebiten.Image {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if img, at := s.raster(scale); img != nil && at == scale {
			return img
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("no raster at scale %v arrived", scale)
	return nil
}

func TestNewVectorSpriteRejectsWhatItCannotDraw(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		svg  string
		size image.Point
	}{
		{name: "not a drawing", svg: "hello", size: image.Pt(10, 10)},
		{name: "no size", svg: redSquare, size: image.Pt(0, 10)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if s, err := NewVectorSprite([]byte(tc.svg), tc.size); err == nil {
				t.Errorf("NewVectorSprite returned %v, want an error", s)
			}
		})
	}
}

func TestVectorSpriteRasterisesAtTheScaleAsked(t *testing.T) {
	t.Parallel()
	s, err := NewVectorSprite([]byte(redSquare), image.Pt(20, 10))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Bounds(); got != image.Rect(0, 0, 20, 10) {
		t.Fatalf("Bounds = %v, want the logical size 20x10", got)
	}

	// The first request starts a raster and has nothing to return yet
	if img, _ := s.raster(2); img != nil {
		t.Fatal("a raster was ready before any had been drawn")
	}
	first := waitRaster(t, s, 2)
	if got := first.Bounds().Size(); got != image.Pt(40, 20) {
		t.Errorf("the raster is %v, want 40x20 at scale 2", got)
	}
	if again, _ := s.raster(2); again != first {
		t.Error("the raster was drawn again for the scale it was already at")
	}

	// A new scale keeps the old raster until the new one is ready
	if img, at := s.raster(3); img != first || at != 2 {
		t.Errorf("mid-way, raster = %v at %v, want the scale 2 raster", img, at)
	}
	second := waitRaster(t, s, 3)
	if got := second.Bounds().Size(); got != image.Pt(60, 30) {
		t.Errorf("the raster is %v, want 60x30 at scale 3", got)
	}
}

func TestRenderedSpriteIsDrawnAtTheScaleAsked(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var scales []float64
	s := NewRenderedSprite(image.Pt(7, 13), func(scale float64) *image.RGBA {
		mu.Lock()
		scales = append(scales, scale)
		mu.Unlock()
		return image.NewRGBA(image.Rectangle{Max: RasterSize(image.Pt(7, 13), scale)})
	})
	if got := s.Bounds(); got != image.Rect(0, 0, 7, 13) {
		t.Fatalf("Bounds = %v, want the logical size 7x13", got)
	}
	img := waitRaster(t, s, 2.5)
	if got := img.Bounds().Size(); got != image.Pt(18, 33) {
		t.Errorf("the rendering is %v, want 18x33 at scale 2.5", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(scales) != 1 || scales[0] != 2.5 {
		t.Errorf("render was called with %v, want once with 2.5", scales)
	}
}

func TestRenderSVGStretchesTheDrawingToFill(t *testing.T) {
	t.Parallel()
	img, err := RenderSVG([]byte(redSquare), image.Pt(8, 4))
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Size(); got != image.Pt(8, 4) {
		t.Fatalf("rendered %v, want 8x4", got)
	}
	red := color.RGBA{0xff, 0, 0, 0xff}
	for _, p := range []image.Point{{0, 0}, {7, 3}, {4, 2}} {
		if got := img.RGBAAt(p.X, p.Y); got != red {
			t.Errorf("pixel %v = %v, want red", p, got)
		}
	}

	if _, err := RenderSVG([]byte("hello"), image.Pt(8, 4)); err == nil {
		t.Error("rendering something that is not a drawing did not fail")
	}
	if _, err := RenderSVG([]byte(redSquare), image.Pt(0, 4)); err == nil {
		t.Error("rendering at no size did not fail")
	}
}

// TestRenderSVGScalesStrokesWithTheShapes guards the court cards' faces,
// which are drawn in fine lines. The rasteriser fits a drawing's shapes to
// the size asked but strokes them at the width the drawing states, so left
// uncorrected a card a quarter of its drawn size has lines four times too
// heavy and its line work fills in solid.
func TestRenderSVGScalesStrokesWithTheShapes(t *testing.T) {
	t.Parallel()
	for _, size := range []int{20, 100, 400} {
		img, err := RenderSVG([]byte(thinLine), image.Pt(size, size))
		if err != nil {
			t.Fatal(err)
		}
		// The ink is the coverage summed over every pixel, which for a
		// line drawn to scale is a fiftieth of the drawing at any size,
		// give or take the anti-aliasing along its edges.
		var ink float64
		for y := range size {
			for x := range size {
				ink += float64(img.RGBAAt(x, y).A) / 0xff
			}
		}
		want := float64(size*size) / 50
		if ink < want*0.85 || ink > want*1.15 {
			t.Errorf("at %dx%d the line covers %.1f pixels, want about %.1f", size, size, ink, want)
		}
	}
}
