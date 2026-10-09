package ui

import (
	"image"
	"image/color"

	"github.com/thomaslaurenson/cellmate/internal/gfx"
)

// kingArt is cellmate's own king, drawn facing right, one string per row.
// It watches the cursor from between the free cells and the home cells, a
// nod to the king in the Windows game without copying its picture.
var kingArt = []string{
	".......K...K...K......",
	"......KYK.KYK.KYK.....",
	"......KYYKYYYKYYK.....",
	"......KYYYYRYYYYK.....",
	"......KYBYYYYYBYK.....",
	"......KKKKKKKKKKK.....",
	"......KSSSSSSSSSK.....",
	"......KSSSSSSSKSK.....",
	"......KSSSSSSSSSKK....",
	"......KSSSSSSSSSSK....",
	"......KSSSSSSSSSK.....",
	".....KWWSSSSSSSWWK....",
	".....KWWWWKKKWWWWK....",
	".....KWWWWWWWWWWWK....",
	"......KWWWWWWWWWK.....",
	"...KKKKKWWWWWWWKKKKK..",
	"..KRRRRRKWWWWWKRRRRRK.",
	".KRRRRRRRKWWWKRRRRRRRK",
	".KRRRWRRRRKKKRRRRWRRRK",
	".KRRRRRRRRRRRRRRRRRRRK",
	".KRRRRRRRRYRRRRRRRRRRK",
	".KRRRRRRRRRRRRRRRRRRRK",
	".KRRRRRRRRYRRRRRRRRRRK",
	".KKKKKKKKKKKKKKKKKKKKK",
}

var kingPalette = map[byte]color.RGBA{
	'K': {0x00, 0x00, 0x00, 0xff},
	'Y': {0xff, 0xd7, 0x00, 0xff},
	'R': {0xc0, 0x00, 0x00, 0xff},
	'B': {0x00, 0x60, 0xff, 0xff},
	'S': {0xff, 0xd0, 0xa8, 0xff},
	'W': {0xff, 0xff, 0xff, 0xff},
}

// kingSize is the sprite's width and height in pixels.
var kingSize = image.Pt(len(kingArt[0]), len(kingArt))

// newKingSprite renders kingArt into a sprite, transparent where the map
// has a dot.
func newKingSprite() *gfx.Sprite {
	img := image.NewRGBA(image.Rectangle{Max: kingSize})
	for y, row := range kingArt {
		for x := range len(row) {
			if c, ok := kingPalette[row[x]]; ok {
				img.SetRGBA(x, y, c)
			}
		}
	}
	return gfx.NewSprite(img)
}

// drawKing draws the king centred on centre at the given scale, mirrored
// when he looks left.
func (a *art) drawKing(dst surface, centre image.Point, scale float64, lookLeft bool) {
	dst.BlitScaled(a.king, centre, scale, lookLeft)
}
