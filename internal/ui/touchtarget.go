package ui

import "image"

// touchMargin is how far outside a control a finger may land and still count,
// in board pixels.
//
// The board is drawn at 640 by 480 and scaled down to fit a phone, so a
// control that is comfortable with a mouse can be a few millimetres across
// under a thumb: the Options box's checkboxes come out at eight screen pixels
// on a modern phone, against minimums of forty-four from Apple and forty-eight
// from Google. Growing the area that answers a tap, rather than the picture,
// keeps every pixel where Windows drew it.
//
// This only helps a control with space around it. A row of menu items tiles
// the dropdown with no gaps, so widening one would take the tap from its
// neighbour; those are left alone and stay small.
const touchMargin = 12

// grow returns r expanded by m on every side. A margin of zero returns r
// unchanged, which is what a mouse gets.
func grow(r image.Rectangle, m int) image.Rectangle {
	if m <= 0 {
		return r
	}
	return r.Inset(-m)
}

// growDown returns r extended downwards by m, and no other way.
//
// The menu bar sits along the top of the board with the free cells directly
// under it and its titles side by side, so down is the only direction with
// anywhere to go. It takes a sliver off the top of the two leftmost cells,
// which are ninety-six pixels tall and lose a few.
func growDown(r image.Rectangle, m int) image.Rectangle {
	if m <= 0 {
		return r
	}
	r.Max.Y += m
	return r
}

// margin returns the slack a tick's input earns. A mouse is exact and gets
// none; a finger gets touchMargin.
func (in input) margin() int {
	if in.touch {
		return touchMargin
	}
	return 0
}
