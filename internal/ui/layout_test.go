package ui

import (
	"image"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/deal"
	"github.com/thomaslaurenson/cellmate/internal/game"
)

// TestCardIsTheShapeItIsDrawn keeps the card slot and the drawings in step.
// The faces are made in a 2 by 3 viewBox and are drawn to fill the slot, so
// a slot of any other shape stretches every one of them.
func TestCardIsTheShapeItIsDrawn(t *testing.T) {
	t.Parallel()
	if cardW*3 != cardH*2 {
		t.Errorf("the card slot is %dx%d, want two by three", cardW, cardH)
	}
}

func TestCellRect(t *testing.T) {
	t.Parallel()
	if got := cellRect(0); got.Min != (image.Point{tableLeft, menuH}) {
		t.Errorf("first free cell at %v, want the table's left edge", got.Min)
	}
	if got := cellRect(7); got.Max.X != tableLeft+tableW {
		t.Errorf("last home cell ends at x=%d, want the table's right edge %d", got.Max.X, tableLeft+tableW)
	}
	// The row is centred, so what it leaves at one edge it leaves at the other.
	if left, right := cellRect(0).Min.X, screenW-cellRect(7).Max.X; left != right {
		t.Errorf("the cells leave %d at the left and %d at the right", left, right)
	}
	if cellRect(3).Max.X > kingRect().Min.X || kingRect().Max.X > cellRect(4).Min.X {
		t.Error("king overlaps the cells")
	}
}

func TestColumnX(t *testing.T) {
	t.Parallel()
	left := columnX(0)
	right := screenW - (columnX(7) + cardW)
	if left != tableLeft || right != left {
		t.Errorf("margins are %d and %d, want %d each", left, right, tableLeft)
	}
	// The columns and the cells above them stand on the same table.
	if left != cellRect(0).Min.X {
		t.Errorf("the first column starts at %d and the first cell at %d", left, cellRect(0).Min.X)
	}
}

// TestHitTestCoversTheTable checks that every point across the board's width
// falls in some column, so there is no dead strip between the piles for a
// click to be lost in.
func TestHitTestCoversTheTable(t *testing.T) {
	t.Parallel()
	for x := range screenW {
		pos, ok := hitTest(image.Pt(x, colTop+1))
		if !ok {
			t.Fatalf("x=%d is in no column", x)
		}
		if want := min(x/(cardW+colGap), deal.Columns-1); pos.Index != want {
			t.Errorf("x=%d is column %d, want %d", x, pos.Index, want)
		}
	}
}

func TestStackStep(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		cards int
		want  int
	}{
		{name: "empty", cards: 0, want: maxStep},
		{name: "fresh deal", cards: 7, want: maxStep},
		{name: "long run squeezes", cards: 19, want: 13},
		{name: "never below minimum", cards: 60, want: minStep},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := stackStep(tc.cards); got != tc.want {
				t.Errorf("stackStep(%d) = %d, want %d", tc.cards, got, tc.want)
			}
		})
	}
}

func TestStackFitsBoard(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 20; n++ {
		bottom := colTop + (n-1)*stackStep(n) + cardH
		if bottom > screenH {
			t.Errorf("a column of %d cards ends at y=%d, past the board", n, bottom)
		}
	}
}

func TestHitTest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pt   image.Point
		want game.Pos
		ok   bool
	}{
		{name: "first free cell", pt: cellRect(0).Min.Add(image.Pt(5, 5)), want: game.Pos{Area: game.FreeCell, Index: 0}, ok: true},
		{name: "last home cell", pt: cellRect(7).Max.Sub(image.Pt(5, 5)), want: game.Pos{Area: game.Home, Index: 3}, ok: true},
		// A cell is hit on its own face. The row stands on the table rather
		// than running to the board's edge, so the margin beside it is table.
		{name: "margin beside the cells", pt: image.Pt(cellRect(0).Min.X-4, menuH+5), ok: false},
		{name: "king", pt: kingRect().Min, ok: false},
		{name: "menu bar", pt: image.Pt(5, 5), ok: false},
		{name: "column face", pt: image.Pt(columnX(3)+30, colTop+200), want: game.Pos{Area: game.Column, Index: 3}, ok: true},
		{name: "just left of a column", pt: image.Pt(columnX(3)-2, colTop+5), want: game.Pos{Area: game.Column, Index: 3}, ok: true},
		{name: "gap between cells and columns", pt: image.Pt(30, colTop-5), ok: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := hitTest(tc.pt)
			if ok != tc.ok || (ok && got != tc.want) {
				t.Errorf("hitTest(%v) = %+v, %v; want %+v, %v", tc.pt, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestCardIndexAt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		y, n int
		want int
	}{
		{name: "empty column", y: colTop + 5, n: 0, want: -1},
		{name: "above the pile", y: colTop - 1, n: 7, want: -1},
		{name: "top card", y: colTop + 1, n: 7, want: 0},
		{name: "third card", y: colTop + 2*maxStep + 1, n: 7, want: 2},
		{name: "exposed card face", y: colTop + 6*maxStep + 50, n: 7, want: 6},
		{name: "below the pile", y: colTop + 6*maxStep + cardH + 1, n: 7, want: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := cardIndexAt(tc.y, tc.n); got != tc.want {
				t.Errorf("cardIndexAt(%d, %d) = %d, want %d", tc.y, tc.n, got, tc.want)
			}
		})
	}
}

func TestGlyphIndex(t *testing.T) {
	t.Parallel()
	if glyphIndex('A') == glyphIndex('B') {
		t.Error("A and B share a glyph")
	}
	if glyphIndex('B') != glyphIndex('A')+1 {
		t.Error("letters are not consecutive in the strip")
	}
	if got, want := glyphIndex('\u2603'), glyphIndex('?'); got != want {
		t.Errorf("a missing rune drew glyph %d, want the question mark %d", got, want)
	}
	for _, r := range "Cards Left: 52 <>%.,!?#" {
		if i := glyphIndex(r); i < 0 || i >= glyphCount {
			t.Errorf("glyph %d for %q is outside the %d-glyph strip", i, r, glyphCount)
		}
	}
}
