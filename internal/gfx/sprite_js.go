//go:build js && wasm

package gfx

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"syscall/js"
)

// Sprite is a source image to blit from, backed by its own canvas element.
// It is either a raster, drawn at the canvas's scale with each of its pixels
// kept whole, or a sprite drawn afresh at whatever scale the canvas settles
// on so that it stays sharp at any size: a vector drawing, which the browser
// draws, or a strip of text set from a font, which the game draws itself.
type Sprite struct {
	// el holds the pixels the sprite is drawn from, elW by elH of them. For
	// a sprite drawn to scale that is its latest rendering, made at scale
	// at.
	el       js.Value
	ctx      js.Value
	elW, elH int
	at       float64
	// w and h are the sprite's size in logical pixels.
	w, h int

	// tints holds recolourings of this sprite, keyed by packed RGBA. Only
	// the glyph strip is tinted, in the few colours text is drawn in, so
	// this never grows past a handful.
	tints map[uint32]js.Value

	// A vector sprite keeps its drawing as an image element, which the
	// browser decodes and draws at any size asked of it. Until the image
	// has loaded there is nothing to draw.
	img    js.Value
	vector bool
	loaded bool

	// render draws a rendered sprite at a scale, as an image the sprite
	// uploads into its canvas.
	render func(scale float64) *image.RGBA
}

// newSurface returns a canvas element of the given size and its 2D context.
func newSurface(w, h int) (js.Value, js.Value) {
	el := js.Global().Get("document").Call("createElement", "canvas")
	el.Set("width", w)
	el.Set("height", h)
	return el, el.Call("getContext", "2d")
}

// NewSprite copies img into a sprite.
func NewSprite(img image.Image) *Sprite {
	b := img.Bounds()
	s := &Sprite{w: b.Dx(), h: b.Dy(), at: 1}
	rgba, ok := img.(*image.RGBA)
	if !ok || rgba.Bounds().Min != (image.Point{}) {
		rgba = image.NewRGBA(image.Rectangle{Max: b.Size()})
		draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	}
	s.upload(rgba)
	return s
}

// NewRenderedSprite returns a sprite of size logical pixels that render
// draws afresh for each scale the canvas settles on. render is given the
// scale and returns the sprite at RasterSize(size, scale) pixels.
func NewRenderedSprite(size image.Point, render func(scale float64) *image.RGBA) *Sprite {
	return &Sprite{w: size.X, h: size.Y, render: render}
}

// NewVectorSprite returns a sprite of an SVG drawing, shown at size logical
// pixels. The drawing is stretched to that size whatever its own shape, so
// only a size of the drawing's own shape shows it undistorted.
//
// The browser decodes the drawing itself, so no SVG parser is linked into
// the binary. It also means a drawing the browser cannot read stays blank
// rather than failing here; the desktop build, which parses the same files,
// is where a bad one is caught.
func NewVectorSprite(svg []byte, size image.Point) (*Sprite, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("vector sprite of size %v", size)
	}
	s := &Sprite{w: size.X, h: size.Y, vector: true}

	buf := js.Global().Get("Uint8Array").New(len(svg))
	js.CopyBytesToJS(buf, svg)
	blob := js.Global().Get("Blob").New([]any{buf}, map[string]any{"type": "image/svg+xml"})
	s.img = js.Global().Get("Image").New()
	// The handler stays registered for the life of the page. Releasing it
	// from inside its own call is not safe while it is still running.
	s.img.Call("addEventListener", "load", js.FuncOf(func(js.Value, []js.Value) any {
		s.loaded = true
		return nil
	}))
	s.img.Set("src", js.Global().Get("URL").Call("createObjectURL", blob))
	return s, nil
}

// Bounds reports the sprite's extent in logical pixels.
func (s *Sprite) Bounds() image.Rectangle { return image.Rect(0, 0, s.w, s.h) }

// scalable reports whether the sprite is drawn afresh for each scale rather
// than being a raster.
func (s *Sprite) scalable() bool { return s.vector || s.render != nil }

// raster makes sure a scalable sprite's pixels are its rendering at scale,
// and reports whether there is anything to draw yet. The browser draws a
// vector sprite at the size it is asked for, so redrawing it into a larger
// canvas is all a sharper rendering takes; a rendered sprite draws itself.
func (s *Sprite) raster(scale float64) bool {
	if s.at == scale {
		return true
	}
	if s.render != nil {
		s.upload(s.render(scale))
	} else {
		if !s.loaded {
			return false
		}
		size := RasterSize(image.Pt(s.w, s.h), scale)
		s.resize(size.X, size.Y)
		s.ctx.Call("drawImage", s.img, 0, 0, size.X, size.Y)
	}
	s.at = scale
	s.tints = nil
	return true
}

// resize gives the sprite's canvas w by h pixels, making it if need be.
func (s *Sprite) resize(w, h int) {
	if s.el.IsUndefined() {
		s.el, s.ctx = newSurface(w, h)
	} else {
		s.el.Set("width", w)
		s.el.Set("height", h)
	}
	s.elW, s.elH = w, h
}

// upload puts img's pixels into the sprite's canvas, sized to fit them.
func (s *Sprite) upload(img *image.RGBA) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	s.resize(w, h)
	pix := make([]byte, w*h*4)
	for y := range h {
		copy(pix[y*w*4:(y+1)*w*4], img.Pix[y*img.Stride:y*img.Stride+w*4])
	}
	buf := js.Global().Get("Uint8Array").New(len(pix))
	js.CopyBytesToJS(buf, pix)
	clamped := js.Global().Get("Uint8ClampedArray").New(buf.Get("buffer"))
	data := js.Global().Get("ImageData").New(clamped, w, h)
	s.ctx.Call("putImageData", data, 0, 0)
}

// tinted returns a copy of the sprite multiplied by c, built once and kept
// until the sprite's pixels change.
func (s *Sprite) tinted(c color.Color) js.Value {
	r, g, b, a := c.RGBA()
	key := uint32(r>>8)<<24 | uint32(g>>8)<<16 | uint32(b>>8)<<8 | uint32(a>>8)
	if v, ok := s.tints[key]; ok {
		return v
	}
	if s.tints == nil {
		s.tints = make(map[uint32]js.Value)
	}
	el, ctx := newSurface(s.elW, s.elH)
	ctx.Call("drawImage", s.el, 0, 0)
	// Painting through the sprite's own alpha keeps the mask's shape and
	// replaces only its colour.
	ctx.Set("globalCompositeOperation", "source-in")
	ctx.Set("fillStyle", cssColour(c))
	ctx.Call("fillRect", 0, 0, s.elW, s.elH)
	s.tints[key] = el
	return el
}

// cssColour renders c as a CSS colour with straight rather than
// premultiplied alpha, which is what a canvas context expects.
func cssColour(c color.Color) string {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return "rgba(0,0,0,0)"
	}
	r = r * 0xffff / a
	g = g * 0xffff / a
	b = b * 0xffff / a
	return "rgba(" + strconv.Itoa(int(r>>8)) + "," + strconv.Itoa(int(g>>8)) + "," +
		strconv.Itoa(int(b>>8)) + "," + strconv.FormatFloat(float64(a)/0xffff, 'f', 3, 64) + ")"
}
