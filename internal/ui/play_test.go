package ui

import (
	"image"
	"strings"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/game"
	"github.com/thomaslaurenson/cellmate/internal/gfx"
	"github.com/thomaslaurenson/cellmate/internal/store"
)

// Deal 1, for reference, with the exposed card last in each column:
//
//	0: JD KD 2S 4C 3S 6D 6S    4: 5D AD JS 4H 8H 6C
//	1: 2D KC KS 5C TD 8S 9C    5: 7H QC AS AC 2C 3D
//	2: 9H 9S 9D TS 4S 8D 2H    6: 7C KH AH 4D JH 8C
//	3: JC 5S QD QH TH QS 6H    7: 5H 3H 3C 7S 7D TC
func newTestTable(t *testing.T, ed edition.Edition, deal int) (*table, *[]string) {
	t.Helper()
	var titles []string
	tb := newTable(Options{Edition: ed, Deal: deal, Saved: store.Defaults()},
		func(s string) { titles = append(titles, s) })
	tb.randIntN = func(int) int { return 616 }
	return tb, &titles
}

func colPt(i int) image.Point  { return image.Pt(columnX(i)+cardW/2, colTop+10) }
func cellPt(i int) image.Point { return cellRect(i).Min.Add(image.Pt(10, 10)) }

func click(tb *table, pt image.Point) {
	tb.step(input{cursor: pt, leftPressed: true})
	tb.step(input{cursor: pt, leftReleased: true})
}

// idle lets time pass, long enough to break a double-click and land any
// flying cards.
func idle(tb *table, ticks int) {
	for range ticks {
		tb.step(input{})
	}
}

func top(tb *table, i int) string { return tb.g.TopAt(game.Pos{Area: game.Column, Index: i}).String() }

func TestNewTableTitle(t *testing.T) {
	t.Parallel()
	_, titles := newTestTable(t, edition.WinXP, 617)
	if len(*titles) != 1 || (*titles)[0] != "cellmate Game #617" {
		t.Errorf("titles = %v", *titles)
	}
}

func TestSelectAndMoveToFreeCell(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	click(tb, colPt(0))
	if tb.selected == nil || *tb.selected != (game.Pos{Area: game.Column, Index: 0}) {
		t.Fatalf("selected = %v, want column 0", tb.selected)
	}
	idle(tb, doubleClickTicks+1)
	click(tb, cellPt(2))
	if got := tb.g.Cell(2).String(); got != "6S" {
		t.Errorf("free cell 2 holds %s, want 6S", got)
	}
	if tb.selected != nil {
		t.Error("selection survived the move")
	}
}

func TestClickingTheSameColumnLaterDeselects(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	click(tb, colPt(0))
	idle(tb, doubleClickTicks+1)
	click(tb, colPt(0))
	if tb.selected != nil || tb.g.Cell(0) != card.None {
		t.Error("a slow second click should only deselect")
	}
}

func TestDoubleClickSendsToFreeCell(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	click(tb, colPt(2))
	click(tb, colPt(2))
	if got := tb.g.Cell(0).String(); got != "2H" {
		t.Errorf("free cell 0 holds %s, want 2H", got)
	}
}

func TestIllegalMoveShowsMessage(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	click(tb, colPt(1))
	idle(tb, doubleClickTicks+1)
	click(tb, colPt(0))
	if tb.dialog == nil || !strings.Contains(strings.Join(tb.dialog.lines, " "), "not allowed") {
		t.Fatalf("dialog = %+v, want the not-allowed message", tb.dialog)
	}
	click(tb, colPt(3))
	if tb.dialog == nil {
		t.Fatal("a click on the board dismissed the modal dialog")
	}
	b := tb.dialog.buttonRect(0)
	click(tb, b.Min.Add(image.Pt(5, 5)))
	if tb.dialog != nil {
		t.Error("OK did not close the dialog")
	}
	if top(tb, 1) != "9C" || top(tb, 0) != "6S" {
		t.Error("the refused move changed the board")
	}
}

func TestDialogButtonNeedsPressAndReleaseOnIt(t *testing.T) {
	t.Parallel()
	picked := -1
	d := newDialog("t", "m", []string{"Yes", "No"}, func(i int) { picked = i })
	yes := d.buttonRect(0).Min.Add(image.Pt(3, 3))
	no := d.buttonRect(1).Min.Add(image.Pt(3, 3))
	d.update(input{cursor: yes, leftPressed: true})
	if d.update(input{cursor: no, leftReleased: true}) || picked != -1 {
		t.Fatal("releasing over a different button picked something")
	}
	d.update(input{cursor: no, leftPressed: true})
	if !d.update(input{cursor: no, leftReleased: true}) || picked != 1 {
		t.Errorf("picked = %d, want 1", picked)
	}
	if !d.update(input{enter: true}) || picked != 0 {
		t.Errorf("Enter picked %d, want the first button", picked)
	}
	if !d.update(input{escape: true}) || picked != 1 {
		t.Errorf("Escape picked %d, want the last button", picked)
	}
}

func TestAutoplayFliesCardsHome(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	click(tb, colPt(5)) // 3D
	idle(tb, doubleClickTicks+1)
	click(tb, cellPt(0))
	click(tb, colPt(5)) // 2C
	idle(tb, doubleClickTicks+1)
	click(tb, cellPt(1))
	if tb.flight == nil {
		t.Fatal("exposing the ace of clubs started no flight")
	}
	click(tb, colPt(0))
	if tb.selected != nil {
		t.Error("input was accepted while a card was flying")
	}
	idle(tb, 10*flightTicks)
	if tb.flight != nil {
		t.Fatal("cards are still flying")
	}
	home := map[string]bool{}
	for i := range game.Slots {
		home[tb.g.HomeTop(i).String()] = true
	}
	if !home["2C"] {
		t.Errorf("home cells hold %v, want the clubs up to 2C", home)
	}
	if tb.g.Cell(1) != card.None {
		t.Error("the two of clubs is still in its free cell")
	}
}

func TestUndoOnlyInXP(t *testing.T) {
	t.Parallel()
	for _, ed := range edition.All {
		tb, _ := newTestTable(t, ed, 1)
		click(tb, colPt(0))
		idle(tb, doubleClickTicks+1)
		click(tb, cellPt(0))
		tb.step(input{key: gfx.KeyF10})
		undone := tb.g.Cell(0) == card.None
		if undone != ed.HasUndo() {
			t.Errorf("edition %s: undone = %v", ed, undone)
		}
	}
}

func TestEscapeDeselects(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	click(tb, colPt(0))
	tb.step(input{escape: true})
	if tb.selected != nil {
		t.Error("Escape left the card selected")
	}
}

func TestNewGameKey(t *testing.T) {
	t.Parallel()
	tb, titles := newTestTable(t, edition.WinXP, 1)
	tb.step(input{key: gfx.KeyF2})
	if tb.g.Number() != 617 {
		t.Errorf("new game is %d, want the stubbed 617", tb.g.Number())
	}
	if last := (*titles)[len(*titles)-1]; last != "cellmate Game #617" {
		t.Errorf("title = %q", last)
	}
}

func TestRevealBuriedCard(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	pt := image.Pt(columnX(0)+10, colTop+1)
	tb.step(input{cursor: pt, rightDown: true})
	if tb.reveal == nil || tb.reveal.index != 0 {
		t.Fatalf("reveal = %+v, want the top card of column 0", tb.reveal)
	}
	tb.step(input{cursor: pt})
	if tb.reveal != nil {
		t.Error("the card stayed revealed after the button was released")
	}
	exposed := image.Pt(columnX(0)+10, colTop+6*maxStep+20)
	tb.step(input{cursor: exposed, rightDown: true})
	if tb.reveal != nil {
		t.Error("revealing the exposed card, which is already visible")
	}
}

func TestTooManyMessageNamesTheLimit(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	tb.showError(&game.TooManyError{Need: 5, Max: 2})
	if tb.dialog == nil {
		t.Fatal("no message shown")
	}
	msg := strings.Join(tb.dialog.lines, " ")
	for _, want := range []string{"5 cards", "only 2"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q does not mention %q", msg, want)
		}
	}
}

func parseCards(t *testing.T, s string) []card.Card {
	t.Helper()
	var out []card.Card
	for _, f := range strings.Fields(s) {
		if f == "--" {
			out = append(out, card.None)
			continue
		}
		c, err := card.Parse(f)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, c)
	}
	return out
}

// setBoard replaces the table's game with one built from text.
func setBoard(t *testing.T, tb *table, cells, homes string, cols ...string) {
	t.Helper()
	var s game.State
	copy(s.Cells[:], parseCards(t, cells))
	copy(s.Homes[:], parseCards(t, homes))
	for i, c := range cols {
		s.Columns[i] = parseCards(t, c)
	}
	tb.g = game.FromState(tb.g.Number(), s)
}

func TestWinOffersAnotherGame(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	setBoard(t, tb, "-- -- -- --", "QC QD QH QS", "KC", "KD", "KH", "KS")
	tb.afterMove()
	idle(tb, 10*flightTicks)
	if !tb.g.Won() {
		t.Fatal("the kings did not all go home")
	}
	if tb.winTick == 0 || tb.dialog != nil {
		t.Fatal("the king did not celebrate before the message")
	}
	idle(tb, winTicks)
	if tb.dialog == nil || !strings.Contains(strings.Join(tb.dialog.lines, " "), "you win") {
		t.Fatalf("dialog = %+v, want the win message", tb.dialog)
	}
	tb.step(input{enter: true})
	if tb.g.Number() != 617 || tb.g.Won() {
		t.Errorf("Yes did not deal a new game: game %d, won %v", tb.g.Number(), tb.g.Won())
	}
}

func TestStuckGameIsReportedOnce(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.Win95, 1)
	setBoard(t, tb, "4H 3H 2D 7C", "AS AC -- --", "5H", "9H", "TD", "KC", "KC", "KC", "KC", "KC")
	tb.changed = true
	tb.step(input{})
	if tb.dialog == nil || !strings.Contains(strings.Join(tb.dialog.lines, " "), "no more legal moves") {
		t.Fatalf("dialog = %+v, want the no-moves message", tb.dialog)
	}
	if strings.Contains(strings.Join(tb.dialog.lines, " "), "F10") {
		t.Error("the 95 edition offered undo")
	}
	tb.step(input{escape: true})
	tb.changed = true
	tb.step(input{})
	if tb.dialog != nil {
		t.Error("the same dead end was reported twice")
	}
}

func TestEmptyColumnAsksHowMany(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		button int
		moved  int
	}{
		{name: "move column", button: 0, moved: 3},
		{name: "move single card", button: 1, moved: 1},
		{name: "cancel", button: 2, moved: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tb, _ := newTestTable(t, edition.WinXP, 1)
			setBoard(t, tb, "-- -- -- --", "-- -- -- --",
				"KD 9C 8H 7S", "", "QC", "QC", "QC", "QC", "QC", "QC")
			click(tb, colPt(0))
			idle(tb, doubleClickTicks+1)
			click(tb, colPt(1))
			if tb.dialog == nil || tb.dialog.title != "Move to Empty Column" {
				t.Fatalf("dialog = %+v, want the empty column question", tb.dialog)
			}
			click(tb, tb.dialog.buttonRect(tc.button).Min.Add(image.Pt(4, 4)))
			if got := len(tb.g.ColumnCards(1)); got != tc.moved {
				t.Errorf("moved %d cards, want %d", got, tc.moved)
			}
		})
	}
}
