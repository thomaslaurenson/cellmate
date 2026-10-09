package gfx

import (
	"image"
	"math"
)

// settleFrames is how many frames the scale must hold still before vector
// sprites are rasterised for it: a quarter of a second at 60 frames a
// second. A window being dragged to a new size changes scale every frame,
// and rasterising the whole deck for each one would stall the drag.
const settleFrames = 15

// view maps the board, which is a fixed number of logical pixels, onto the
// surface it is shown on. The board is scaled to fit the surface, whatever
// its shape, centred across it and set at its top, so the surface has an
// empty margin below the board or either side of it unless it happens to be
// the board's own shape. The top is where a window's menu bar belongs, so a
// tall surface has its whole margin under the table rather than half of it
// above the menu.
//
// Every drawing operation goes through the view, which is what lets the game
// keep its 640 by 480 coordinates while the surface is any size at all.
// Vector sprites are rasterised at the scale the view has settled on, which
// lags the scale itself while the surface is being resized.
type view struct {
	board image.Point
	// box is the surface, in its own pixels.
	box image.Point
	// scale and offX map a logical point p onto the surface at
	// (offX+p.X*scale, p.Y*scale).
	scale float64
	offX  float64
	// settled is the scale vector sprites are drawn for: the first scale the
	// view saw, and after that any scale that has held for settleFrames.
	settled float64
	// still counts the frames the scale has been unchanged for.
	still int
}

// resize fits the board to a surface of w by h pixels. It is called once a
// frame, whether or not the surface changed, which is what counts the
// frames a new scale has held still for.
func (v *view) resize(w, h int) {
	v.box = image.Pt(w, h)
	scale, offX := fit(v.board, float64(w), float64(h))
	if scale == v.scale {
		v.still++
	} else {
		v.still = 0
	}
	v.scale, v.offX = scale, offX
	if scale > 0 && (v.settled == 0 || v.still >= settleFrames) {
		v.settled = scale
	}
}

// fit returns the scale that fits board into a box of boxW by boxH, and how
// far in from the box's left edge the board's top left corner lands once the
// board is centred across the box. The corner is at the top of the box. A
// box or a board with no size fits nothing, which is reported as a scale of
// zero.
func fit(board image.Point, boxW, boxH float64) (scale, offX float64) {
	if boxW <= 0 || boxH <= 0 || board.X <= 0 || board.Y <= 0 {
		return 0, 0
	}
	scale = min(boxW/float64(board.X), boxH/float64(board.Y))
	offX = (boxW - float64(board.X)*scale) / 2
	return scale, offX
}

// point maps a logical point onto the surface.
func (v *view) point(p image.Point) image.Point {
	return image.Pt(round(v.offX+float64(p.X)*v.scale), round(float64(p.Y)*v.scale))
}

// rect maps a logical rectangle onto the surface. Each edge is rounded on
// its own, so two rectangles that meet on the board meet on the surface too,
// with neither a gap nor an overlap between them.
func (v *view) rect(r image.Rectangle) image.Rectangle {
	return image.Rectangle{Min: v.point(r.Min), Max: v.point(r.Max)}
}

// logical maps a position on the surface back onto the board. A position in
// the margin maps to a point off the board, which the game reads as nothing
// being under the pointer. Before the first resize everything maps to the
// origin.
func (v *view) logical(x, y float64) image.Point {
	if v.scale == 0 {
		return image.Point{}
	}
	return image.Pt(
		int(math.Floor((x-v.offX)/v.scale)),
		int(math.Floor(y/v.scale)),
	)
}

// RasterSize is the size in pixels of a sprite of size logical pixels drawn
// at scale, rounded up so it is never short of the space it is drawn into.
// It is what a sprite's render function is expected to return.
func RasterSize(size image.Point, scale float64) image.Point {
	return image.Pt(
		max(int(math.Ceil(float64(size.X)*scale)), 1),
		max(int(math.Ceil(float64(size.Y)*scale)), 1),
	)
}

// scaleRect scales r by s, rounding each edge to the nearest pixel.
func scaleRect(r image.Rectangle, s float64) image.Rectangle {
	return image.Rect(
		round(float64(r.Min.X)*s), round(float64(r.Min.Y)*s),
		round(float64(r.Max.X)*s), round(float64(r.Max.Y)*s),
	)
}

func round(f float64) int { return int(math.Round(f)) }
