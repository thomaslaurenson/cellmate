package ui

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/thomaslaurenson/cellmate/internal/gfx"
)

// The board is lettered in Go Mono, set at the scale of the window so the
// text is as sharp as the cards. Every glyph is drawn into a cell of glyphW
// by glyphH logical pixels, the measure of the 7 by 13 bitmap font the board
// was laid out on, so text takes the same room wherever it is drawn and
// however large the window is.
//
// The strip holds the printable ASCII range, one glyph to a cell in order,
// and stands in the question mark for anything else.
const (
	glyphW     = 7
	glyphH     = 13
	glyphFirst = ' '
	glyphCount = '~' - ' ' + 1
)

// typeface is the parsed font, shared by every strip. The font may be used
// from several goroutines at once; the faces made from it may not, so each
// rendering makes its own.
var typeface = func() *sfnt.Font {
	f, err := opentype.Parse(gomono.TTF)
	if err != nil {
		panic("embedded font does not parse: " + err.Error())
	}
	return f
}()

// glyphIndex returns which glyph of the strip draws r.
func glyphIndex(r rune) int {
	if r < glyphFirst || r >= glyphFirst+glyphCount {
		r = '?'
	}
	return int(r - glyphFirst)
}

// newGlyphStrip returns the strip as a sprite, drawn afresh at each scale
// the window settles on.
func newGlyphStrip() *gfx.Sprite {
	return gfx.NewRenderedSprite(image.Pt(glyphW, glyphH*glyphCount), renderGlyphs)
}

// renderGlyphs draws the strip at scale: white glyphs on a clear ground, one
// to a cell, for the canvas to tint. Each cell's edges are rounded the way
// the canvas rounds them when it cuts a glyph out again.
func renderGlyphs(scale float64) *image.RGBA {
	cellW, cellH := glyphW*scale, glyphH*scale
	face := newFace(cellW, cellH)
	m := face.Metrics()
	ascent, descent := fixedToFloat(m.Ascent), fixedToFloat(m.Descent)

	img := image.NewRGBA(image.Rectangle{Max: gfx.RasterSize(image.Pt(glyphW, glyphH*glyphCount), scale)})
	d := &font.Drawer{Dst: img, Src: image.White, Face: face}
	for i := range glyphCount {
		r := rune(glyphFirst + i)
		top := math.Round(float64(i) * cellH)
		bottom := math.Round(float64(i+1) * cellH)
		advance, _ := face.GlyphAdvance(r)
		// The glyph sits in the middle of its cell, and its baseline
		// leaves the same room above the ascent as below the descent
		x := (cellW - fixedToFloat(advance)) / 2
		y := top + ((bottom-top)-(ascent+descent))/2 + ascent
		d.Dot = fixed.Point26_6{X: floatToFixed(x), Y: floatToFixed(math.Round(y))}
		d.DrawString(string(r))
	}
	return img
}

// newFace returns the font at the largest size whose glyphs fit a cell of w
// by h pixels: wide enough for its widest glyph, and tall enough for its
// ascent and descent together.
func newFace(w, h float64) font.Face {
	const probeSize = 100
	probe := mustFace(probeSize)
	advance, _ := probe.GlyphAdvance('M')
	m := probe.Metrics()
	perAdvance := fixedToFloat(advance) / probeSize
	perHeight := fixedToFloat(m.Ascent+m.Descent) / probeSize
	return mustFace(min(w/perAdvance, h/perHeight))
}

// mustFace returns the font at size pixels. The font is part of the binary
// and parsed at start-up, so a face that cannot be made is a bug.
func mustFace(size float64) font.Face {
	f, err := opentype.NewFace(typeface, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic("font face: " + err.Error())
	}
	return f
}

func fixedToFloat(v fixed.Int26_6) float64 { return float64(v) / 64 }

func floatToFixed(v float64) fixed.Int26_6 { return fixed.Int26_6(math.Round(v * 64)) }

// text draws s with its top-left corner at x, y.
func (a *art) text(dst surface, s string, x, y int, c color.Color) {
	for k, r := range []rune(s) {
		if r == ' ' {
			continue
		}
		i := glyphIndex(r)
		dst.BlitTinted(a.glyphs,
			image.Rect(0, i*glyphH, glyphW, (i+1)*glyphH),
			image.Pt(x+k*glyphW, y), c)
	}
}
