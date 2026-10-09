//go:build js && wasm

package gfx

import (
	"image"
	"image/color"
	"syscall/js"
)

// Canvas is the surface the game draws onto.
type Canvas struct {
	ctx  js.Value
	view view
}

// newCanvas returns a canvas drawing through ctx and showing a board of the
// given logical size.
func newCanvas(ctx js.Value, board image.Point) *Canvas {
	return &Canvas{ctx: ctx, view: view{board: board}}
}

// begin readies the canvas for a frame on a bitmap of w by h pixels.
func (c *Canvas) begin(w, h int) { c.view.resize(w, h) }

// Fill paints the whole canvas with col, margins included.
func (c *Canvas) Fill(col color.Color) {
	c.fillRect(image.Rectangle{Max: c.view.box}, col, "source-over")
}

// FillRect paints r with col.
func (c *Canvas) FillRect(r image.Rectangle, col color.Color) {
	c.fillRect(c.view.rect(r), col, "source-over")
}

// InvertRect replaces r with its negative, which is how Windows FreeCell
// showed a selected card.
func (c *Canvas) InvertRect(r image.Rectangle) {
	// Compositing white in difference mode yields 1-destination, which is
	// the inversion. Drawing an inverted copy of the region instead would
	// need a read back off the GPU every frame.
	c.fillRect(c.view.rect(r), color.White, "difference")
}

// fillRect paints a rectangle of the bitmap, in its own pixels.
func (c *Canvas) fillRect(r image.Rectangle, col color.Color, mode string) {
	c.ctx.Set("globalCompositeOperation", mode)
	c.ctx.Set("fillStyle", cssColour(col))
	c.ctx.Call("fillRect", r.Min.X, r.Min.Y, r.Dx(), r.Dy())
}

// Blit draws the part of src inside srcRect with its top-left corner at at.
func (c *Canvas) Blit(src *Sprite, srcRect image.Rectangle, at image.Point) {
	c.blit(src, srcRect, at, nil)
}

// BlitTinted draws srcRect of src at at, multiplied by col. It is meant for
// white masks such as the font's glyph strip, which the caller colours.
func (c *Canvas) BlitTinted(src *Sprite, srcRect image.Rectangle, at image.Point, col color.Color) {
	c.blit(src, srcRect, at, col)
}

// blit draws srcRect of src at at, tinted by col unless it is nil.
func (c *Canvas) blit(src *Sprite, srcRect image.Rectangle, at image.Point, col color.Color) {
	px, ok := c.source(src)
	if !ok {
		return
	}
	el := src.el
	if col != nil {
		el = src.tinted(col)
	}
	sr := srcRect
	if src.scalable() {
		// A sprite drawn to scale has more pixels than it has logical
		// ones, so the part asked for is scaled to match
		sr = scaleRect(srcRect, src.at)
	}
	var dr image.Rectangle
	if src.scalable() && px == 1 {
		// The raster was made for this scale, so it goes down pixel for
		// pixel and nothing is resampled
		dr = image.Rectangle{Min: c.view.point(at)}
		dr.Max = dr.Min.Add(sr.Size())
	} else {
		dr = c.view.rect(image.Rectangle{Min: at, Max: at.Add(srcRect.Size())})
	}
	c.ctx.Set("globalCompositeOperation", "source-over")
	// A raster's pixels are kept whole; a drawing's stale raster, shown
	// while a fresh one is made, is smoothed as it is stretched
	c.ctx.Set("imageSmoothingEnabled", src.scalable() && px != 1)
	c.ctx.Call("drawImage", el,
		sr.Min.X, sr.Min.Y, sr.Dx(), sr.Dy(),
		dr.Min.X, dr.Min.Y, dr.Dx(), dr.Dy())
}

// BlitScaled draws the whole of src centred on centre, scaled by scale and
// mirrored left to right when mirror is set.
func (c *Canvas) BlitScaled(src *Sprite, centre image.Point, scale float64, mirror bool) {
	px, ok := c.source(src)
	if !ok {
		return
	}
	w, h := float64(src.elW)*px*scale, float64(src.elH)*px*scale
	dst := c.view.point(centre)
	c.ctx.Set("globalCompositeOperation", "source-over")
	c.ctx.Set("imageSmoothingEnabled", src.scalable())
	c.ctx.Call("save")
	c.ctx.Call("translate", dst.X, dst.Y)
	if mirror {
		c.ctx.Call("scale", -1, 1)
	}
	c.ctx.Call("drawImage", src.el, -w/2, -h/2, w, h)
	c.ctx.Call("restore")
}

// source readies src for drawing and returns how many bitmap pixels each of
// its pixels covers: the view's scale for a raster, and for a drawing 1 once
// its raster matches the view, or a little either side while a fresh raster
// is on its way. A drawing that has not loaded yet has nothing to draw.
func (c *Canvas) source(src *Sprite) (float64, bool) {
	if !src.scalable() {
		return c.view.scale, true
	}
	if !src.raster(c.view.settled) {
		return 0, false
	}
	return c.view.scale / src.at, true
}
