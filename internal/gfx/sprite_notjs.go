//go:build !(js && wasm)

package gfx

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// Sprite is a source image to blit from. It is either a raster, drawn at the
// canvas's scale with each of its pixels kept whole, or a sprite drawn
// afresh at whatever scale the canvas settles on so that it stays sharp at
// any size: a vector drawing, or a strip of text set from a font.
type Sprite struct {
	// img is the image the sprite is drawn from. For a sprite drawn to
	// scale it is the latest rendering, made at scale at, and nil until
	// the first one is ready. Both belong to the main thread.
	img *ebiten.Image
	at  float64

	// The rest belongs to a sprite drawn to scale. render draws it off the
	// main thread, because a whole deck of cards takes most of a second on
	// one core, into an ordinary image that the main thread turns into an
	// Ebitengine image on the next blit. mu guards the hand-over.
	render  func(scale float64) *image.RGBA
	size    image.Point
	mu      sync.Mutex
	ready   *image.RGBA
	readyAt float64
	// busy is the scale being drawn now, or zero. One rendering runs at a
	// time, since render need not be safe to call twice at once.
	busy float64
}

// NewSprite copies img into a sprite.
func NewSprite(img image.Image) *Sprite {
	return &Sprite{img: ebiten.NewImageFromImage(img), at: 1}
}

// NewRenderedSprite returns a sprite of size logical pixels that render
// draws afresh for each scale the canvas settles on. render is given the
// scale and returns the sprite at RasterSize(size, scale) pixels. It runs
// off the main thread, one call at a time.
func NewRenderedSprite(size image.Point, render func(scale float64) *image.RGBA) *Sprite {
	return &Sprite{render: render, size: size}
}

// NewVectorSprite returns a sprite of an SVG drawing, shown at size logical
// pixels. The drawing is stretched to that size whatever its own shape, so
// only a size of the drawing's own shape shows it undistorted.
func NewVectorSprite(svg []byte, size image.Point) (*Sprite, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("vector sprite of size %v", size)
	}
	icon, err := parseSVG(svg)
	if err != nil {
		return nil, err
	}
	return NewRenderedSprite(size, func(scale float64) *image.RGBA {
		return renderSVG(icon, RasterSize(size, scale))
	}), nil
}

// Bounds reports the sprite's extent in logical pixels.
func (s *Sprite) Bounds() image.Rectangle {
	if s.scalable() {
		return image.Rectangle{Max: s.size}
	}
	return s.img.Bounds()
}

// scalable reports whether the sprite is drawn afresh for each scale rather
// than being a raster.
func (s *Sprite) scalable() bool { return s.render != nil }

// raster returns the image to draw a scalable sprite from at scale, and the
// scale that image was made at: the one asked for once a rendering at it is
// ready, and the most recent one until then. It starts the rendering it
// lacks and returns nil while there has never been one. It is called from
// the main thread, which is where Ebitengine images are made and freed.
func (s *Sprite) raster(scale float64) (*ebiten.Image, float64) {
	s.mu.Lock()
	if s.ready != nil {
		if s.img != nil {
			s.img.Deallocate()
		}
		s.img, s.at = ebiten.NewImageFromImage(s.ready), s.readyAt
		s.ready = nil
	}
	start := s.at != scale && s.busy == 0
	if start {
		s.busy = scale
	}
	s.mu.Unlock()
	if start {
		go s.rasterise(scale)
	}
	return s.img, s.at
}

// rasterise draws the sprite at scale and hands the result to the main
// thread.
func (s *Sprite) rasterise(scale float64) {
	img := s.render(scale)
	s.mu.Lock()
	s.ready, s.readyAt, s.busy = img, scale, 0
	s.mu.Unlock()
}

// RenderSVG rasterises an SVG drawing at the given size in pixels, stretched
// to fill it. It is what a vector sprite is drawn from, on its own for checks
// and tools.
func RenderSVG(svg []byte, size image.Point) (*image.RGBA, error) {
	icon, err := parseSVG(svg)
	if err != nil {
		return nil, err
	}
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("render svg at size %v", size)
	}
	return renderSVG(icon, size), nil
}

// parseSVG reads a drawing, refusing one with anything in it the rasteriser
// cannot draw rather than drawing it without. Text that is not a drawing at
// all parses as an empty one with no size, so that is refused too.
func parseSVG(svg []byte) (*oksvg.SvgIcon, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg), oksvg.StrictErrorMode)
	if err != nil {
		return nil, fmt.Errorf("parse svg: %w", err)
	}
	if icon.ViewBox.W <= 0 || icon.ViewBox.H <= 0 {
		return nil, errors.New("parse svg: not a drawing with a size")
	}
	return icon, nil
}

// renderSVG draws icon into a new image of the given size, stretching the
// drawing to fill it.
//
// oksvg fits the drawing's shapes to the size but strokes each path at the
// width the drawing states, in the drawing's own units, however far the
// shapes have been scaled. Fitted into a card a quarter of its drawn size, a
// two-unit line came out two pixels wide rather than half a pixel, and the
// fine line work of a court card's face filled in solid. So the strokes are
// scaled here along with the shapes, on a copy of the paths, which leaves
// the parsed drawing as read for the next scale it is asked for.
func renderSVG(icon *oksvg.SvgIcon, size image.Point) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Max: size})
	drawing := *icon
	drawing.SVGPaths = strokesToScale(icon.SVGPaths,
		float64(size.X)/icon.ViewBox.W, float64(size.Y)/icon.ViewBox.H)
	drawing.SetTarget(0, 0, float64(size.X), float64(size.Y))
	scanner := rasterx.NewScannerGV(size.X, size.Y, img, img.Bounds())
	drawing.Draw(rasterx.NewDasher(size.X, size.Y, scanner), 1)
	return img
}

// strokesToScale returns a copy of paths with every length of their strokes,
// the width and any dash pattern, scaled to match shapes scaled by scaleX
// and scaleY. Stretched by different amounts each way, a line has no one
// right width; the geometric mean of the two keeps its area of ink right.
func strokesToScale(paths []oksvg.SvgPath, scaleX, scaleY float64) []oksvg.SvgPath {
	k := math.Sqrt(scaleX * scaleY)
	out := make([]oksvg.SvgPath, len(paths))
	for i, p := range paths {
		p.LineWidth *= k
		p.DashOffset *= k
		if len(p.Dash) > 0 {
			dash := make([]float64, len(p.Dash))
			for j, d := range p.Dash {
				dash[j] = d * k
			}
			p.Dash = dash
		}
		out[i] = p
	}
	return out
}
