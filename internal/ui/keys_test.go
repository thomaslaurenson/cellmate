package ui

import (
	"errors"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/game"
)

func typeKeys(tb *table, s string) {
	for _, r := range s {
		tb.step(input{chars: []rune{r}})
	}
}

func TestKeyboardToFreeCellAndBack(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	typeKeys(tb, "10")
	if got := tb.g.Cell(0).String(); got != "6S" {
		t.Fatalf("1 then 0 put %s in free cell 0, want 6S", got)
	}
	typeKeys(tb, "30")
	typeKeys(tb, "0")
	if tb.selected == nil || *tb.selected != (game.Pos{Area: game.FreeCell, Index: 0}) {
		t.Fatalf("0 selected %v, want free cell 0", tb.selected)
	}
	typeKeys(tb, "0")
	if *tb.selected != (game.Pos{Area: game.FreeCell, Index: 1}) {
		t.Fatalf("second 0 selected %v, want free cell 1", tb.selected)
	}
	typeKeys(tb, "0")
	if *tb.selected != (game.Pos{Area: game.FreeCell, Index: 0}) {
		t.Fatalf("third 0 selected %v, want to wrap to free cell 0", tb.selected)
	}
	typeKeys(tb, "4")
	if tb.dialog == nil {
		t.Fatal("6S onto 6C was not refused")
	}
}

func TestKeyboardColumnMoves(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	setBoard(t, tb, "-- -- -- --", "AS -- -- --", "9H 2S", "TS", "KC", "KC", "KC", "KC", "KC", "KC")
	typeKeys(tb, "19")
	if tb.g.HomeTop(0).String() != "2S" {
		t.Fatalf("1 then 9 did not send 2S home: home holds %v", tb.g.HomeTop(0))
	}
	typeKeys(tb, "12")
	if top(tb, 1) != "9H" {
		t.Errorf("1 then 2 left column 2 showing %s, want 9H", top(tb, 1))
	}
	typeKeys(tb, "11")
	if tb.selected != nil {
		t.Error("pressing the same column twice left it selected")
	}
	typeKeys(tb, "9")
	if tb.selected != nil {
		t.Error("9 on its own selected something")
	}
	typeKeys(tb, "x0")
	if tb.selected != nil {
		t.Error("0 with every free cell empty selected something")
	}
}

func TestKeyboardFreeCellsFull(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	setBoard(t, tb, "2D 3D 4D 5D", "-- -- -- --", "9H", "TS")
	typeKeys(tb, "10")
	if tb.dialog == nil || !strings.Contains(strings.Join(tb.dialog.lines, " "), "not allowed") {
		t.Errorf("dialog = %+v, want a refusal", tb.dialog)
	}
}

func TestEmptyColumnHotkeys(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		key   rune
		moved int
	}{{'c', 3}, {'S', 1}} {
		tb, _ := newTestTable(t, edition.WinXP, 1)
		setBoard(t, tb, "-- -- -- --", "-- -- -- --",
			"KD 9C 8H 7S", "", "QC", "QC", "QC", "QC", "QC", "QC")
		typeKeys(tb, "12")
		typeKeys(tb, string(tc.key))
		if got := len(tb.g.ColumnCards(1)); got != tc.moved {
			t.Errorf("key %q moved %d cards, want %d", tc.key, got, tc.moved)
		}
	}
}

func TestDefaultHotkeys(t *testing.T) {
	t.Parallel()
	got := defaultHotkeys([]string{"Yes", "No"})
	if got[0] != 'y' || got[1] != 'n' {
		t.Errorf("Yes/No hotkeys = %q", got)
	}
	got = defaultHotkeys([]string{"Move column", "Move single card", "Cancel"})
	if got[0] != 0 || got[1] != 0 || got[2] != 'c' {
		t.Errorf("clashing labels got hotkeys %q", got)
	}
}

func TestKingArtIsRectangular(t *testing.T) {
	t.Parallel()
	for i, row := range kingArt {
		if len(row) != kingSize.X {
			t.Errorf("row %d is %d wide, want %d", i, len(row), kingSize.X)
		}
		for _, px := range []byte(row) {
			if _, ok := kingPalette[px]; !ok && px != '.' {
				t.Errorf("row %d uses %q, which has no colour", i, px)
			}
		}
	}
	if kingSize.X > kingGap || kingSize.Y > cardH {
		t.Errorf("king %v does not fit between the cells", kingSize)
	}
}

func TestWinKing(t *testing.T) {
	t.Parallel()
	start, s0 := winKing(0)
	if start != kingRect().Min.Add(kingRect().Size().Div(2)) || s0 != 1 {
		t.Errorf("the celebration starts at %v scale %v, not where the king sits", start, s0)
	}
	end, s1 := winKing(winTicks)
	if end != image.Pt(screenW/2, (colTop+screenH)/2) || math.Abs(s1-6) > 1e-9 {
		t.Errorf("the celebration ends at %v scale %v", end, s1)
	}
	if _, s := winKing(10 * winTicks); s != s1 {
		t.Error("the king kept growing after the animation ended")
	}
}

func TestWinWithSaveErrorShowsBoth(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.WinXP)
	fs.err = errors.New("disk full")
	tb.settings.QuickPlay = true
	setBoard(t, tb, "-- -- -- --", "QC QD QH QS", "KC", "KD", "KH", "KS")
	tb.afterMove()
	tb.step(input{})
	if !strings.Contains(dialogText(tb), "disk full") {
		t.Fatalf("dialog = %q, want the save error first", dialogText(tb))
	}
	tb.step(input{enter: true})
	if !strings.Contains(dialogText(tb), "you win") {
		t.Errorf("after the save error, dialog = %q, want the win message", dialogText(tb))
	}
}

func TestSelectedCellCardDrawsWhereItIs(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	typeKeys(tb, "10")
	typeKeys(tb, "0")
	if tb.g.TopAt(*tb.selected) != card.New(6, card.Spades) {
		t.Error("the selected free cell does not hold the six of spades")
	}
}
