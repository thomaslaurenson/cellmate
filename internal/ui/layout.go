package ui

import (
	"image"

	"github.com/thomaslaurenson/cellmate/internal/deal"
	"github.com/thomaslaurenson/cellmate/internal/game"
)

// The board is drawn at a fixed logical size of 640 by 480 and scaled to the
// window, so every coordinate here is in that space.
const (
	screenW = 640
	screenH = 480

	menuH = 20

	// cardW and cardH are two by three, which is the shape the card faces
	// are drawn in. The original game's cards were 71 by 96 and the faces
	// were stretched sideways to fill them; a slot the drawings' own shape
	// fits them instead, so the pips are round and the court cards are the
	// shape they were drawn.
	cardW = 64
	cardH = 96

	// colTop is where the tableau starts, one card height plus a gap below
	// the cells.
	colTop = menuH + cardH + 16

	// maxStep is the vertical offset between stacked cards when a column has
	// room; minStep is the tightest it squeezes to before running off the
	// bottom of the board.
	maxStep = 18
	minStep = 6
)

// The table is the eight tableau columns and the gaps between them, centred
// on the board. The cards are narrower than the original game's, and the
// width that leaves goes to the gaps rather than the margins, so the table
// spans the 640 by 480 board the way the original's does.
const (
	colGap    = 16
	tableW    = 8*cardW + 7*colGap
	tableLeft = (screenW - tableW) / 2
)

// kingGap is the space between the free cells and the home cells where the
// king sits. The row of eight cells spans the same table as the columns
// below it, so the cells stay at its edges as they do in the original game
// and what is left in the middle is the king's.
const kingGap = tableW - 8*cardW

// cellRect returns the rectangle of top-row cell i: free cells are 0 to 3
// from the left, home cells 4 to 7.
func cellRect(i int) image.Rectangle {
	x := tableLeft + i*cardW
	if i >= 4 {
		x += kingGap
	}
	return image.Rect(x, menuH, x+cardW, menuH+cardH)
}

// kingRect returns the square the king is drawn in, centred between the
// free cells and the home cells.
func kingRect() image.Rectangle {
	const size = 32
	x := tableLeft + 4*cardW + (kingGap-size)/2
	y := menuH + (cardH-size)/2
	return image.Rect(x, y, x+size, y+size)
}

// columnX returns the left edge of tableau column i.
func columnX(i int) int {
	return tableLeft + i*(cardW+colGap)
}

// stackStep returns the vertical offset between the n cards of a column,
// shrinking it so the last card stays on the board.
func stackStep(n int) int {
	if n <= 1 {
		return maxStep
	}
	avail := screenH - colTop - cardH - 4
	step := avail / (n - 1)
	return max(minStep, min(maxStep, step))
}

// cardPoint returns where card j of an n-card column i is drawn.
func cardPoint(i, j, n int) image.Point {
	return image.Pt(columnX(i), colTop+j*stackStep(n))
}

// slotPoint returns where the exposed card of a board position is drawn,
// for a column holding n cards.
func slotPoint(p game.Pos, n int) image.Point {
	switch p.Area {
	case game.FreeCell:
		return cellRect(p.Index).Min
	case game.Home:
		return cellRect(game.Slots + p.Index).Min
	default:
		return cardPoint(p.Index, max(n-1, 0), n)
	}
}

// hitTest returns the board position under a point. Each column owns the
// full-height strip including half the gap on either side, so a click just
// beside a pile still counts.
func hitTest(pt image.Point) (game.Pos, bool) {
	if pt.Y >= menuH && pt.Y < menuH+cardH {
		for i := range 2 * game.Slots {
			if pt.In(cellRect(i)) {
				if i < game.Slots {
					return game.Pos{Area: game.FreeCell, Index: i}, true
				}
				return game.Pos{Area: game.Home, Index: i - game.Slots}, true
			}
		}
		return game.Pos{}, false
	}
	if pt.Y < colTop || pt.Y >= screenH {
		return game.Pos{}, false
	}
	for i := range deal.Columns {
		left := columnX(i) - colGap/2
		if pt.X >= left && pt.X < left+cardW+colGap {
			return game.Pos{Area: game.Column, Index: i}, true
		}
	}
	return game.Pos{}, false
}

// cardIndexAt returns which of the n cards in a column is showing at height
// y, for revealing a buried card, or -1 when y is past the pile.
func cardIndexAt(y, n int) int {
	if n == 0 || y < colTop {
		return -1
	}
	step := stackStep(n)
	j := (y - colTop) / step
	if j >= n-1 {
		if y < colTop+(n-1)*step+cardH {
			return n - 1
		}
		return -1
	}
	return j
}
