package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/deal"
	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/game"
	"github.com/thomaslaurenson/cellmate/internal/gfx"
)

// drawOp is one call made against a recorder, kept as a value so a test can
// assert on what was drawn rather than on the pixels it would have produced.
type drawOp struct {
	op     string
	rect   image.Rectangle
	at     image.Point
	colour color.Color
	sprite *gfx.Sprite
	scale  float64
	mirror bool
}

// recorder is a surface that keeps every drawing call instead of painting
// it, so the table can be drawn in a test with no window and no GPU.
type recorder struct {
	ops []drawOp
}

var _ surface = (*recorder)(nil)

func (r *recorder) Fill(c color.Color) {
	r.ops = append(r.ops, drawOp{op: "Fill", colour: c})
}

func (r *recorder) FillRect(rect image.Rectangle, c color.Color) {
	r.ops = append(r.ops, drawOp{op: "FillRect", rect: rect, colour: c})
}

func (r *recorder) InvertRect(rect image.Rectangle) {
	r.ops = append(r.ops, drawOp{op: "InvertRect", rect: rect})
}

func (r *recorder) Blit(src *gfx.Sprite, srcRect image.Rectangle, at image.Point) {
	r.ops = append(r.ops, drawOp{op: "Blit", sprite: src, rect: srcRect, at: at})
}

func (r *recorder) BlitTinted(src *gfx.Sprite, srcRect image.Rectangle, at image.Point, c color.Color) {
	r.ops = append(r.ops, drawOp{op: "BlitTinted", sprite: src, rect: srcRect, at: at, colour: c})
}

func (r *recorder) BlitScaled(src *gfx.Sprite, centre image.Point, scale float64, mirror bool) {
	r.ops = append(r.ops, drawOp{op: "BlitScaled", sprite: src, at: centre, scale: scale, mirror: mirror})
}

// count reports how many calls of the named operation were made.
func (r *recorder) count(op string) int {
	n := 0
	for _, o := range r.ops {
		if o.op == op {
			n++
		}
	}
	return n
}

// only returns the single call of the named operation, failing when there is
// not exactly one.
func (r *recorder) only(t *testing.T, op string) drawOp {
	t.Helper()
	var found []drawOp
	for _, o := range r.ops {
		if o.op == op {
			found = append(found, o)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s was called %d times, want exactly 1", op, len(found))
	}
	return found[0]
}

// newTestArt builds the sprites the table draws with, failing the test if
// any card face cannot be loaded.
func newTestArt(t *testing.T) *art {
	t.Helper()
	a, err := newArt()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// draw renders the table into a recorder.
func draw(t *testing.T, tb *table) *recorder {
	t.Helper()
	r := &recorder{}
	newTestArt(t).drawTable(r, tb)
	return r
}

func TestDrawTableStartsWithTheTableColour(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	r := draw(t, tb)

	if len(r.ops) == 0 {
		t.Fatal("drawTable drew nothing")
	}
	first := r.ops[0]
	if first.op != "Fill" {
		t.Fatalf("first call was %s, want Fill so the table is cleared before anything is drawn on it", first.op)
	}
	if first.colour != color.Color(tableGreen) {
		t.Errorf("table filled with %v, want %v", first.colour, tableGreen)
	}
}

func TestDrawTableDrawsEveryDealtCard(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	r := draw(t, tb)

	// Every card on the table is a blit of its own face, so a fresh deal
	// draws 52 different sprites.
	faces := make(map[*gfx.Sprite]bool)
	for _, o := range r.ops {
		if o.op == "Blit" && o.rect.Size() == image.Pt(cardW, cardH) {
			faces[o.sprite] = true
		}
	}
	if len(faces) != card.Count {
		t.Errorf("drew %d different card faces, want %d", len(faces), card.Count)
	}
}

func TestDrawTableInvertsOnlyTheSelectedCard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		selected *game.Pos
		want     int
	}{
		{name: "nothing selected", selected: nil, want: 0},
		{name: "a column selected", selected: &game.Pos{Area: game.Column, Index: 0}, want: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tb, _ := newTestTable(t, edition.WinXP, 1)
			tb.selected = tc.selected
			r := draw(t, tb)

			if got := r.count("InvertRect"); got != tc.want {
				t.Fatalf("InvertRect called %d times, want %d", got, tc.want)
			}
			if tc.want == 0 {
				return
			}
			n := len(tb.g.ColumnCards(tc.selected.Index))
			pt := slotPoint(*tc.selected, n)
			want := image.Rect(pt.X, pt.Y, pt.X+cardW, pt.Y+cardH)
			if got := r.only(t, "InvertRect").rect; got != want {
				t.Errorf("inverted %v, want the selected card at %v", got, want)
			}
		})
	}
}

func TestDrawTableWatchesTheCursorWithTheKing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		cursor     image.Point
		wantMirror bool
	}{
		{name: "cursor on the left", cursor: image.Pt(10, 300), wantMirror: true},
		{name: "cursor on the right", cursor: image.Pt(screenW-10, 300), wantMirror: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tb, _ := newTestTable(t, edition.WinXP, 1)
			tb.cursor = tc.cursor
			r := draw(t, tb)

			king := r.only(t, "BlitScaled")
			if king.mirror != tc.wantMirror {
				t.Errorf("king mirrored = %v, want %v", king.mirror, tc.wantMirror)
			}
			if king.scale != 1 {
				t.Errorf("king drawn at scale %v, want 1 while the game is in play", king.scale)
			}
		})
	}
}

func TestDrawTableBevelsEveryCellAndHome(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	r := draw(t, tb)

	// Each of the free cells and home cells is outlined by a bevel, which is
	// four filled edges.
	for i := range 2 * game.Slots {
		want := cellRect(i)
		top := image.Rect(want.Min.X, want.Min.Y, want.Max.X, want.Min.Y+1)
		found := false
		for _, o := range r.ops {
			if o.op == "FillRect" && o.rect == top && o.colour == color.Color(cellShadow) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("cell %d has no bevelled top edge at %v", i, top)
		}
	}
}

func TestTextDrawsOneGlyphPerVisibleRune(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: "", want: 0},
		{name: "one word", text: "Game", want: 4},
		{name: "spaces are not drawn", text: "a b", want: 2},
		{name: "all spaces", text: "   ", want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			newTestArt(t).text(r, tc.text, 0, 0, darkBlack)

			if got := r.count("BlitTinted"); got != tc.want {
				t.Fatalf("drew %d glyphs for %q, want %d", got, tc.text, tc.want)
			}
		})
	}
}

func TestTextAdvancesPastSpaces(t *testing.T) {
	t.Parallel()
	r := &recorder{}
	newTestArt(t).text(r, "a b", 0, 0, darkBlack)

	if got := len(r.ops); got != 2 {
		t.Fatalf("drew %d glyphs, want 2", got)
	}
	// The space still takes a cell, so the second letter sits two advances
	// along rather than one.
	want := 2 * glyphW
	if got := r.ops[1].at.X - r.ops[0].at.X; got != want {
		t.Errorf("second glyph is %d pixels along, want %d", got, want)
	}
}

// TestMenuBarShowsTheGameNumberWhenAsked counts glyphs: with the option on,
// the bar gains exactly the characters of "Game #1", spaces aside.
func TestMenuBarShowsTheGameNumberWhenAsked(t *testing.T) {
	t.Parallel()
	glyphs := func(on bool) int {
		tb, _ := newTestTable(t, edition.WinXP, 1)
		tb.settings.GameNumber = on
		return draw(t, tb).count("BlitTinted")
	}
	off, on := glyphs(false), glyphs(true)
	if want := off + len("Game#1"); on != want {
		t.Errorf("the bar drew %d glyphs with the game number and %d without, want %d more", on, off, len("Game#1"))
	}
}

func TestDrawTableIsDeterministic(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, deal.Columns)

	// One art for both passes: a second one would hold different sprites,
	// and the calls would differ for that reason rather than this one.
	a := newTestArt(t)
	first, second := &recorder{}, &recorder{}
	a.drawTable(first, tb)
	a.drawTable(second, tb)

	if len(first.ops) != len(second.ops) {
		t.Fatalf("two draws of one board made %d and %d calls", len(first.ops), len(second.ops))
	}
	for i := range first.ops {
		if first.ops[i] != second.ops[i] {
			t.Fatalf("call %d differs between draws: %+v then %+v", i, first.ops[i], second.ops[i])
		}
	}
}
