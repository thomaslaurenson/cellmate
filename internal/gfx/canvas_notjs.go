//go:build !(js && wasm)

package gfx

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Canvas is the surface the game draws onto.
type Canvas struct {
	screen *ebiten.Image
	// pixel is a single white texel, stretched to draw solid rectangles.
	// Scaling it keeps edges pixel-exact where an anti-aliased vector shape
	// would blur them.
	pixel *ebiten.Image
	view  view
}

// newCanvas returns a canvas showing a board of the given logical size. It
// draws onto whatever screen the frame gives it.
func newCanvas(board image.Point) *Canvas {
	px := ebiten.NewImage(1, 1)
	px.Fill(color.White)
	return &Canvas{pixel: px, view: view{board: board}}
}

// begin readies the canvas for a frame drawn onto screen.
func (c *Canvas) begin(screen *ebiten.Image) {
	c.screen = screen
	b := screen.Bounds()
	c.view.resize(b.Dx(), b.Dy())
}

// invert turns a white rectangle into a negative of what is under it, which
// is how Windows FreeCell showed a selected card.
var invert = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorOneMinusDestinationColor,
	BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
	BlendFactorDestinationRGB:   ebiten.BlendFactorZero,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

// Fill paints the whole canvas with col, margins included.
func (c *Canvas) Fill(col color.Color) { c.screen.Fill(col) }

// FillRect paints r with col.
func (c *Canvas) FillRect(r image.Rectangle, col color.Color) {
	c.fillRect(c.view.rect(r), col, ebiten.Blend{})
}

// InvertRect replaces r with its negative.
func (c *Canvas) InvertRect(r image.Rectangle) {
	c.fillRect(c.view.rect(r), color.White, invert)
}

// fillRect paints a rectangle of the screen, in its own pixels.
func (c *Canvas) fillRect(r image.Rectangle, col color.Color, blend ebiten.Blend) {
	if r.Empty() {
		return
	}
	op := &ebiten.DrawImageOptions{Blend: blend}
	op.GeoM.Scale(float64(r.Dx()), float64(r.Dy()))
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	op.ColorScale.ScaleWithColor(col)
	c.screen.DrawImage(c.pixel, op)
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
	img, s, made := c.source(src)
	if img == nil {
		return
	}
	if srcRect != src.Bounds() {
		// A sprite drawn to scale has more pixels than it has logical
		// ones, so the part asked for is scaled to match
		img = img.SubImage(scaleRect(srcRect, made)).(*ebiten.Image)
	}
	op := &ebiten.DrawImageOptions{}
	if col != nil {
		op.ColorScale.ScaleWithColor(col)
	}
	op.GeoM.Scale(s, s)
	if src.scalable() && s != 1 {
		op.Filter = ebiten.FilterLinear
	}
	dst := c.view.point(at)
	op.GeoM.Translate(float64(dst.X), float64(dst.Y))
	c.screen.DrawImage(img, op)
}

// BlitScaled draws the whole of src centred on centre, scaled by scale and
// mirrored left to right when mirror is set.
func (c *Canvas) BlitScaled(src *Sprite, centre image.Point, scale float64, mirror bool) {
	img, s, _ := c.source(src)
	if img == nil {
		return
	}
	b := img.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	if mirror {
		op.GeoM.Scale(-1, 1)
	}
	op.GeoM.Scale(s*scale, s*scale)
	if src.scalable() {
		op.Filter = ebiten.FilterLinear
	}
	dst := c.view.point(centre)
	op.GeoM.Translate(float64(dst.X), float64(dst.Y))
	c.screen.DrawImage(img, op)
}

// source returns the image src is drawn from, how many screen pixels each of
// its pixels covers, and the scale the image was made at. A raster is made
// at 1 and covers the view's scale; a sprite drawn to scale covers 1 once
// its rendering matches the view, or a little either side while a fresh one
// is on its way. A sprite with no rendering yet has nothing to draw.
func (c *Canvas) source(src *Sprite) (img *ebiten.Image, pixel, made float64) {
	if !src.scalable() {
		return src.img, c.view.scale, 1
	}
	img, made = src.raster(c.view.settled)
	if img == nil {
		return nil, 0, 0
	}
	return img, c.view.scale / made, made
}
