//go:build !(js && wasm)

package gfx

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Ebitengine queues drawing commands and only needs a graphics device when a
// frame is presented, so every operation below runs with no window. Reading
// the result back does need one, which is why these tests assert that the
// calls are well formed rather than what they painted. The browser back-end
// is checked against this one by eye outside the test suite.

func TestCanvasOperationsRunAgainstARealImage(t *testing.T) {
	t.Parallel()
	c := newCanvas(image.Pt(64, 48))
	// A screen one and a half times the board, so nothing lands on whole
	// pixels by accident
	c.begin(ebiten.NewImage(96, 72))
	sprite := NewSprite(image.NewRGBA(image.Rect(0, 0, 8, 8)))
	drawing, err := NewVectorSprite([]byte(redSquare), image.Pt(8, 8))
	if err != nil {
		t.Fatal(err)
	}
	waitRaster(t, drawing, c.view.settled)
	fresh, err := NewVectorSprite([]byte(redSquare), image.Pt(8, 8))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		draw func()
	}{
		{name: "fill", draw: func() { c.Fill(color.White) }},
		{name: "fill a rectangle", draw: func() { c.FillRect(image.Rect(1, 1, 5, 5), color.Black) }},
		{name: "fill the whole board", draw: func() { c.FillRect(image.Rect(0, 0, 64, 48), color.Black) }},
		{name: "fill an empty rectangle", draw: func() { c.FillRect(image.Rect(3, 3, 3, 3), color.Black) }},
		{name: "invert", draw: func() { c.InvertRect(image.Rect(2, 2, 4, 4)) }},
		{name: "blit a whole sprite", draw: func() { c.Blit(sprite, sprite.Bounds(), image.Pt(10, 10)) }},
		{name: "blit part of a sprite", draw: func() { c.Blit(sprite, image.Rect(2, 2, 6, 6), image.Pt(0, 0)) }},
		{name: "blit tinted", draw: func() {
			c.BlitTinted(sprite, sprite.Bounds(), image.Pt(20, 20), color.RGBA{0xff, 0, 0, 0xff})
		}},
		{name: "blit scaled", draw: func() { c.BlitScaled(sprite, image.Pt(30, 30), 2, false) }},
		{name: "blit mirrored", draw: func() { c.BlitScaled(sprite, image.Pt(30, 30), 1, true) }},
		{name: "blit at a fractional scale", draw: func() { c.BlitScaled(sprite, image.Pt(30, 30), 0.5, false) }},
		{name: "blit partly off the surface", draw: func() { c.Blit(sprite, sprite.Bounds(), image.Pt(-4, -4)) }},
		{name: "blit a drawing", draw: func() { c.Blit(drawing, drawing.Bounds(), image.Pt(10, 10)) }},
		{name: "blit part of a drawing", draw: func() { c.Blit(drawing, image.Rect(2, 2, 6, 6), image.Pt(0, 0)) }},
		{name: "blit a drawing tinted", draw: func() {
			c.BlitTinted(drawing, drawing.Bounds(), image.Pt(20, 20), color.RGBA{0xff, 0, 0, 0xff})
		}},
		{name: "blit a drawing scaled and mirrored", draw: func() { c.BlitScaled(drawing, image.Pt(30, 30), 2, true) }},
		{name: "blit a drawing with no raster yet", draw: func() { c.Blit(fresh, fresh.Bounds(), image.Pt(0, 0)) }},
		{name: "blit a drawing whose raster is stale", draw: func() {
			// A new screen size before the view has settled on it leaves
			// the drawing's raster at the old scale, stretched to fit
			c.begin(ebiten.NewImage(128, 96))
			c.Blit(drawing, drawing.Bounds(), image.Pt(10, 10))
			c.BlitScaled(drawing, image.Pt(30, 30), 2, false)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// These share one screen, so they cannot run in parallel
			tc.draw()
		})
	}
}

func TestSpriteBoundsMatchTheSourceImage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  image.Rectangle
		want image.Rectangle
	}{
		{name: "from the origin", src: image.Rect(0, 0, 71, 96), want: image.Rect(0, 0, 71, 96)},
		{name: "offset source", src: image.Rect(10, 20, 81, 116), want: image.Rect(0, 0, 71, 96)},
		{name: "single pixel", src: image.Rect(0, 0, 1, 1), want: image.Rect(0, 0, 1, 1)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NewSprite(image.NewRGBA(tc.src)).Bounds(); got != tc.want {
				t.Errorf("Bounds = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLayoutGivesTheGameTheWholeWindow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		factor       float64
		outW, outH   int
		wantW, wantH int
	}{
		{name: "at the board's own size", factor: 1, outW: 640, outH: 480, wantW: 640, wantH: 480},
		{name: "in a larger window", factor: 1, outW: 1920, outH: 1080, wantW: 1920, wantH: 1080},
		{name: "in a tiny window", factor: 1, outW: 100, outH: 50, wantW: 100, wantH: 50},
		{name: "on a high-resolution monitor", factor: 2, outW: 640, outH: 480, wantW: 1280, wantH: 960},
		{name: "a fractional factor rounds up", factor: 1.5, outW: 641, outH: 481, wantW: 962, wantH: 722},
		{name: "with no monitor to ask", factor: 0, outW: 640, outH: 480, wantW: 640, wantH: 480},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := &window{scaleFactor: func() float64 { return tc.factor }}
			gotW, gotH := w.Layout(tc.outW, tc.outH)
			if gotW != tc.wantW || gotH != tc.wantH {
				t.Errorf("Layout(%d, %d) = %d, %d, want %d, %d",
					tc.outW, tc.outH, gotW, gotH, tc.wantW, tc.wantH)
			}
		})
	}
}
