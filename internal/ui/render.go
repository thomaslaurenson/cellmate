package ui

import (
	"image"
	"image/color"

	"github.com/thomaslaurenson/cellmate/internal/gfx"
)

// surface is what the table is drawn onto: the handful of operations the
// board needs, and no more. Declaring it here rather than beside the canvas
// that satisfies it keeps the drawing code independent of the platform, and
// lets a test draw a frame into a recorder instead of a window.
type surface interface {
	Fill(c color.Color)
	FillRect(r image.Rectangle, c color.Color)
	InvertRect(r image.Rectangle)
	Blit(src *gfx.Sprite, srcRect image.Rectangle, at image.Point)
	BlitTinted(src *gfx.Sprite, srcRect image.Rectangle, at image.Point, c color.Color)
	BlitScaled(src *gfx.Sprite, centre image.Point, scale float64, mirror bool)
}

var _ surface = (*gfx.Canvas)(nil)
