package ui

import (
	"errors"
	"image"
	"strings"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/gfx"
	"github.com/thomaslaurenson/cellmate/internal/store"
)

// fakeStore records every save, and can be told to fail.
type fakeStore struct {
	saves []store.Data
	err   error
}

func (f *fakeStore) Save(d store.Data) error {
	f.saves = append(f.saves, d)
	return f.err
}

func (f *fakeStore) last() store.Data { return f.saves[len(f.saves)-1] }

func newStoredTable(t *testing.T, ed edition.Edition) (*table, *fakeStore) {
	t.Helper()
	fs := &fakeStore{}
	tb := newTable(Options{Edition: ed, Deal: 1, Saved: store.Defaults(), Store: fs, CanExit: true}, func(string) {})
	tb.randIntN = func(int) int { return 616 }
	return tb, fs
}

func press(tb *table, k gfx.Key) { tb.step(input{key: k}) }

// pickButton clicks the button of the open dialog with the given label.
func pickButton(t *testing.T, tb *table, label string) {
	t.Helper()
	if tb.dialog == nil {
		t.Fatalf("no dialog open to press %q in", label)
	}
	for i, b := range tb.dialog.buttons {
		if b == label {
			click(tb, tb.dialog.buttonRect(i).Min.Add(image.Pt(4, 4)))
			return
		}
	}
	t.Fatalf("dialog %q has no %q button", tb.dialog.title, label)
}

func dialogText(tb *table) string {
	if tb.dialog == nil {
		return ""
	}
	return tb.dialog.title + ": " + strings.Join(tb.dialog.lines, " ")
}

// makeAMove plays deal 1's six of spades into a free cell.
func makeAMove(tb *table) {
	click(tb, colPt(0))
	idle(tb, doubleClickTicks+1)
	click(tb, cellPt(0))
}

func TestNewGameAsksToResignAndCountsALoss(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.WinXP)
	makeAMove(tb)
	press(tb, gfx.KeyF2)
	if !strings.Contains(dialogText(tb), "resign") {
		t.Fatalf("dialog = %q, want the resign question", dialogText(tb))
	}
	pickButton(t, tb, "No")
	if tb.g.Number() != 1 || len(fs.saves) != 0 {
		t.Fatal("No still left the game")
	}
	press(tb, gfx.KeyF2)
	pickButton(t, tb, "Yes")
	if tb.g.Number() != 617 {
		t.Errorf("game is %d after resigning, want 617", tb.g.Number())
	}
	if len(fs.saves) != 1 || fs.last().Stats.Lost != 1 || tb.session.Lost != 1 {
		t.Errorf("the resigned game was not saved as a loss: %+v", fs.saves)
	}
}

func TestUntouchedGameLeavesWithoutAsking(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.WinXP)
	press(tb, gfx.KeyF2)
	if tb.dialog != nil || tb.g.Number() != 617 || len(fs.saves) != 0 {
		t.Errorf("dialog %q, game %d, saves %d", dialogText(tb), tb.g.Number(), len(fs.saves))
	}
}

func TestWinIsSaved(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.WinXP)
	setBoard(t, tb, "-- -- -- --", "QC QD QH QS", "KC", "KD", "KH", "KS")
	tb.afterMove()
	idle(tb, 10*flightTicks+winTicks)
	if len(fs.saves) != 1 || fs.last().Stats.Won != 1 {
		t.Fatalf("saves = %+v, want one win", fs.saves)
	}
	pickButton(t, tb, "Yes")
	if len(fs.saves) != 1 {
		t.Error("starting the next game after a win recorded something more")
	}
}

func TestStuckGameCountsOnceWhenLeft(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.Win95)
	setBoard(t, tb, "4H 3H 2D 7C", "AS AC -- --", "5H", "9H", "TD", "KC", "KC", "KC", "KC", "KC")
	tb.changed = true
	tb.step(input{})
	pickButton(t, tb, "No")
	press(tb, gfx.KeyF2)
	if tb.dialog != nil {
		t.Errorf("asked %q about a game already over", dialogText(tb))
	}
	if len(fs.saves) != 1 || fs.last().Stats.Lost != 1 {
		t.Errorf("saves = %+v, want one loss", fs.saves)
	}
}

func TestSelectGame(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.Win95)
	press(tb, gfx.KeyF3)
	if tb.dialog == nil || tb.dialog.field == nil || tb.dialog.field.text != "617" {
		t.Fatalf("dialog = %+v, want a number box holding a random deal", tb.dialog)
	}
	tb.step(input{chars: []rune("11a982")})
	if got := tb.dialog.field.text; got != "11982" {
		t.Fatalf("field = %q, want the typing to replace the suggestion, digits only", got)
	}
	tb.step(input{backspace: true})
	tb.step(input{chars: []rune("2")})
	if got := tb.dialog.field.text; got != "11982" {
		t.Fatalf("field = %q after backspace and retyping", got)
	}
	tb.step(input{enter: true})
	if tb.g.Number() != 11982 {
		t.Errorf("game = %d, want 11982", tb.g.Number())
	}
}

func TestSelectGameOutOfRange(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.Win95)
	press(tb, gfx.KeyF3)
	tb.dialog.field.text = "32001"
	tb.step(input{enter: true})
	if !strings.Contains(dialogText(tb), "1 to 32000") {
		t.Fatalf("dialog = %q, want the range message", dialogText(tb))
	}
	tb.step(input{enter: true})
	if tb.dialog == nil || tb.dialog.field == nil {
		t.Error("the number box did not come back after the range message")
	}
	if tb.g.Number() != 1 {
		t.Error("an out of range number changed the game")
	}
}

func TestSelectGameFieldLimit(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.Win95)
	press(tb, gfx.KeyF3)
	tb.step(input{chars: []rune("9999999")})
	if got := tb.dialog.field.text; got != "99999" {
		t.Errorf("field holds %q, want five digits, as many as 32000 needs", got)
	}
}

func TestSelectGameBackspaceClearsSuggestion(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.Win95)
	press(tb, gfx.KeyF3)
	tb.step(input{backspace: true})
	if got := tb.dialog.field.text; got != "" {
		t.Errorf("field = %q, want Backspace to clear the selected suggestion", got)
	}
	tb.step(input{enter: true})
	if !strings.Contains(dialogText(tb), "Please enter") {
		t.Errorf("an empty number gave %q", dialogText(tb))
	}
}

func TestRestartGame(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.WinXP)
	makeAMove(tb)
	tb.restartGame()
	pickButton(t, tb, "Yes")
	if tb.g.Number() != 1 || tb.g.CanUndo() {
		t.Errorf("restart gave game %d with undo %v", tb.g.Number(), tb.g.CanUndo())
	}
}

func TestStatistics(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.WinXP)
	tb.total.Win()
	tb.total.Win()
	tb.total.Lose()
	tb.session.Win()
	press(tb, gfx.KeyF4)
	text := dialogText(tb)
	for _, want := range []string{"won 1, lost 0, 100%", "won 2, lost 1, 66%", "Current: 1 loss"} {
		if !strings.Contains(text, want) {
			t.Errorf("statistics %q missing %q", text, want)
		}
	}
	pickButton(t, tb, "Clear")
	pickButton(t, tb, "Yes")
	if tb.total.Won != 0 || tb.session.Won != 0 || fs.last().Stats.Won != 0 {
		t.Error("Clear left statistics behind")
	}
	if !strings.Contains(dialogText(tb), "won 0, lost 0, 0%") {
		t.Errorf("after clearing, dialog = %q", dialogText(tb))
	}
}

func TestOptionsApplyAndSave(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.WinXP)
	press(tb, gfx.KeyF5)
	d := tb.dialog
	if d == nil || len(d.checks) != 6 {
		t.Fatalf("dialog = %+v, want six options", d)
	}
	// Messages off, quick play and the game number on, and the 95 edition
	for _, i := range []int{0, 1, 3, 4} {
		click(tb, d.checkRect(i).Min.Add(image.Pt(2, 2)))
	}
	if d.checks[5].on {
		t.Fatal("choosing 95 left XP chosen too")
	}
	click(tb, d.checkRect(4).Min.Add(image.Pt(2, 2)))
	if !d.checks[4].on {
		t.Fatal("clicking a chosen radio button cleared it")
	}
	pickButton(t, tb, "OK")

	want := store.Settings{Edition: "95", Messages: false, QuickPlay: true, DoubleClick: true, GameNumber: true}
	if tb.settings != want || tb.edition != edition.Win95 {
		t.Errorf("settings = %+v, edition %v", tb.settings, tb.edition)
	}
	if fs.last().Settings != want {
		t.Errorf("saved %+v, want %+v", fs.last().Settings, want)
	}
	if tb.g.Number() != 617 {
		t.Errorf("game %d, want the new edition to start a new game", tb.g.Number())
	}
}

func TestOptionsEditionChangeMidGame(t *testing.T) {
	t.Parallel()
	choose95 := func(t *testing.T, tb *table) {
		t.Helper()
		press(tb, gfx.KeyF5)
		click(tb, tb.dialog.checkRect(4).Min.Add(image.Pt(2, 2)))
		pickButton(t, tb, "OK")
	}

	tb, fs := newStoredTable(t, edition.WinXP)
	tb.newDeal(500000)
	makeAMove(tb)
	if !tb.g.CanUndo() {
		t.Fatal("no move was made")
	}
	choose95(t, tb)
	if !strings.Contains(dialogText(tb), "resign") {
		t.Fatalf("dialog = %q, want the resign question", dialogText(tb))
	}
	pickButton(t, tb, "No")
	if tb.edition != edition.WinXP || tb.g.Number() != 500000 || fs.last().Settings.Edition != "xp" {
		t.Fatalf("No: edition %v, game %d, saved %q; want the XP game untouched",
			tb.edition, tb.g.Number(), fs.last().Settings.Edition)
	}

	choose95(t, tb)
	pickButton(t, tb, "Yes")
	if tb.edition != edition.Win95 || fs.last().Settings.Edition != "95" || fs.last().Stats.Lost != 1 {
		t.Errorf("Yes: edition %v, saved %+v; want 95 saved with a loss", tb.edition, fs.last())
	}
	if err := tb.edition.CheckDeal(tb.g.Number()); err != nil {
		t.Errorf("new game: %v", err)
	}
}

func TestOptionsCancel(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.WinXP)
	press(tb, gfx.KeyF5)
	click(tb, tb.dialog.checkRect(0).Min.Add(image.Pt(2, 2)))
	pickButton(t, tb, "Cancel")
	if !tb.settings.Messages || len(fs.saves) != 0 {
		t.Error("Cancel kept the change")
	}
}

func TestMessagesOff(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.WinXP)
	tb.settings.Messages = false
	click(tb, colPt(1))
	idle(tb, doubleClickTicks+1)
	click(tb, colPt(0))
	if tb.dialog != nil {
		t.Errorf("dialog %q shown with messages off", dialogText(tb))
	}
}

func TestQuickPlaySkipsAnimation(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.WinXP)
	tb.settings.QuickPlay = true
	setBoard(t, tb, "-- -- -- --", "QC QD QH QS", "KC", "KD", "KH", "KS")
	tb.afterMove()
	if tb.flight != nil || !tb.g.Won() {
		t.Errorf("quick play: flight %v, won %v", tb.flight, tb.g.Won())
	}
}

func TestDoubleClickOff(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.WinXP)
	tb.settings.DoubleClick = false
	click(tb, colPt(2))
	click(tb, colPt(2))
	if tb.g.CanUndo() {
		t.Error("a double-click moved a card with the option off")
	}
}

func TestSaveFailureReportedOnce(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.WinXP)
	fs.err = errors.New("disk full")
	tb.recordLoss()
	if !strings.Contains(dialogText(tb), "disk full") {
		t.Fatalf("dialog = %q, want the save error", dialogText(tb))
	}
	tb.dialog = nil
	tb.recordLoss()
	if tb.dialog != nil || len(fs.saves) != 1 {
		t.Error("a second failure was reported or attempted")
	}
}

func TestNoStoreStillCounts(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	tb.recordWin()
	if tb.total.Won != 1 || tb.dialog != nil {
		t.Error("a table with no store should still keep score")
	}
}

func TestMenus(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.WinXP)
	menus := tb.menus()
	title := func(i int) image.Point { return menuTitleRect(menus, i).Min.Add(image.Pt(3, 3)) }
	item := func(i, j int) image.Point { return menuItemRect(menus, i, j).Min.Add(image.Pt(3, 3)) }

	click(tb, title(0))
	if tb.menuOpen != 0 {
		t.Fatal("clicking Game did not open it")
	}
	tb.step(input{cursor: title(1)})
	if tb.menuOpen != 1 {
		t.Fatal("sliding to Help did not switch menus")
	}
	tb.step(input{cursor: title(0)})

	// Restart is greyed out before any move, and clicking it closes the
	// menu without doing anything.
	tb.step(input{cursor: item(0, 2)})
	if tb.menuHover != -1 {
		t.Error("a disabled item was highlighted")
	}
	click(tb, item(0, 2))
	if tb.menuOpen != -1 || tb.dialog != nil {
		t.Error("a disabled item did something")
	}

	click(tb, title(0))
	tb.step(input{cursor: item(0, 4)})
	if tb.menuHover != 4 {
		t.Errorf("hover = %d, want Statistics", tb.menuHover)
	}
	click(tb, item(0, 4))
	if !strings.HasPrefix(dialogText(tb), "Statistics") {
		t.Errorf("dialog = %q, want statistics", dialogText(tb))
	}
	tb.step(input{escape: true})

	click(tb, title(0))
	click(tb, colPt(0))
	if tb.menuOpen != -1 || tb.selected != nil {
		t.Error("a click on the board while a menu was open should only close it")
	}

	click(tb, title(0))
	tb.step(input{escape: true})
	if tb.menuOpen != -1 {
		t.Error("Escape left the menu open")
	}

	click(tb, title(0))
	click(tb, title(0))
	if tb.menuOpen != -1 {
		t.Error("clicking the open title did not close it")
	}
}

func TestMenuContents(t *testing.T) {
	t.Parallel()
	labels := func(tb *table) string {
		var out []string
		for _, it := range tb.menus()[0].items {
			out = append(out, it.label)
		}
		return strings.Join(out, ",")
	}
	xp, _ := newStoredTable(t, edition.WinXP)
	if l := labels(xp); !strings.Contains(l, "Undo") || !strings.Contains(l, "Exit") {
		t.Errorf("XP desktop menu = %s", l)
	}
	w95, _ := newTestTable(t, edition.Win95, 1)
	if l := labels(w95); strings.Contains(l, "Undo") || strings.Contains(l, "Exit") {
		t.Errorf("95 browser menu = %s", l)
	}
}

func TestExit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		exit func(*table)
	}{
		{name: "Exit in the Game menu", exit: (*table).exitGame},
		{name: "closing the window", exit: func(tb *table) { tb.step(input{close: true}) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fresh, fs := newStoredTable(t, edition.WinXP)
			tc.exit(fresh)
			if !fresh.quit || len(fs.saves) != 0 {
				t.Errorf("before any move: quit %v, saves %d; want to quit without a loss",
					fresh.quit, len(fs.saves))
			}

			tb, fs := newStoredTable(t, edition.WinXP)
			makeAMove(tb)
			tc.exit(tb)
			if !strings.Contains(dialogText(tb), "resign") {
				t.Fatalf("dialog = %q, want the resign question", dialogText(tb))
			}
			pickButton(t, tb, "No")
			if tb.quit || len(fs.saves) != 0 {
				t.Fatal("No still left the game")
			}
			tc.exit(tb)
			pickButton(t, tb, "Yes")
			if !tb.quit || fs.last().Stats.Lost != 1 {
				t.Errorf("Yes: quit %v, lost %d; want to quit with a loss", tb.quit, tb.total.Lost)
			}
		})
	}
}

func TestCloseIsIgnoredUnderAMessageBox(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.WinXP)
	press(tb, gfx.KeyF1)
	d := tb.dialog
	tb.step(input{close: true})
	if tb.quit || tb.dialog != d {
		t.Error("closing the window acted behind an open message box")
	}
}

func TestHelpAndAbout(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.WinXP)
	tb.version = "1.2.3"
	press(tb, gfx.KeyF1)
	if !strings.HasPrefix(dialogText(tb), "How to Play") {
		t.Errorf("F1 opened %q", dialogText(tb))
	}
	tb.step(input{enter: true})
	tb.showAbout()
	if !strings.Contains(dialogText(tb), "cellmate 1.2.3") {
		t.Errorf("about = %q", dialogText(tb))
	}
}

func TestDebugBox(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.Win95)
	tb.step(input{key: gfx.KeyF10})
	if tb.dialog != nil {
		t.Fatal("plain F10 in the 95 edition did something")
	}
	tb.step(input{key: gfx.KeyF10, ctrl: true, shift: true})
	pickButton(t, tb, "Abort")
	idle(tb, winTicks+2)
	if !tb.g.Won() || fs.last().Stats.Won != 1 {
		t.Errorf("Abort: won %v, saves %+v", tb.g.Won(), fs.saves)
	}
	pickButton(t, tb, "No")
	press(tb, gfx.KeyF2)

	tb.step(input{key: gfx.KeyF10, ctrl: true, shift: true})
	pickButton(t, tb, "Retry")
	if !strings.Contains(dialogText(tb), "you lose") {
		t.Fatalf("Retry showed %q", dialogText(tb))
	}
	pickButton(t, tb, "Yes")
	if fs.last().Stats.Lost != 1 {
		t.Errorf("Retry then Yes did not record a loss: %+v", fs.last().Stats)
	}

	tb.step(input{key: gfx.KeyF10, ctrl: true, shift: true})
	pickButton(t, tb, "Ignore")
	if tb.dialog != nil || tb.g.Won() {
		t.Error("Ignore changed something")
	}
}

// TestSelectGameTakesTheImpossibleDeals covers the two deals hidden below
// the range the box names: a minus sign is accepted first and nowhere else.
func TestSelectGameTakesTheImpossibleDeals(t *testing.T) {
	t.Parallel()
	tb, _ := newStoredTable(t, edition.Win95)
	press(tb, gfx.KeyF3)
	tb.step(input{chars: []rune("-1-")})
	if got := tb.dialog.field.text; got != "-1" {
		t.Fatalf("field = %q, want -1 with the second minus dropped", got)
	}
	tb.step(input{enter: true})
	if tb.dialog != nil || tb.g.Number() != -1 {
		t.Fatalf("dialog %q, game %d, want game -1", dialogText(tb), tb.g.Number())
	}
	if got := top(tb, 0); got != "KC" {
		t.Errorf("column 1 shows %s, want the king of clubs of deal -1", got)
	}

	press(tb, gfx.KeyF3)
	tb.step(input{chars: []rune("-3")})
	tb.step(input{enter: true})
	if !strings.Contains(dialogText(tb), "1 to 32000") {
		t.Errorf("deal -3 gave %q, want the range message", dialogText(tb))
	}
}

// TestImpossibleDealCountsForNothing: Windows kept the two deals out of the
// statistics, so leaving one mid-way asks nothing and records nothing, and
// nor does the hidden way of winning one.
func TestImpossibleDealCountsForNothing(t *testing.T) {
	t.Parallel()
	tb, fs := newStoredTable(t, edition.WinXP)
	tb.newDeal(-2)
	makeAMove(tb)
	press(tb, gfx.KeyF2)
	if tb.dialog != nil || tb.g.Number() != 617 {
		t.Fatalf("dialog %q, game %d, want to have left without asking", dialogText(tb), tb.g.Number())
	}
	if len(fs.saves) != 0 || tb.session.Lost != 0 {
		t.Errorf("leaving deal -2 recorded %+v", fs.saves)
	}

	tb.newDeal(-1)
	tb.showDebug()
	pickButton(t, tb, "Abort")
	idle(tb, winTicks+2)
	if !strings.Contains(dialogText(tb), "you win") {
		t.Fatalf("dialog = %q, want the win message", dialogText(tb))
	}
	if len(fs.saves) != 0 || tb.session.Won != 0 {
		t.Errorf("winning deal -1 recorded %+v", fs.saves)
	}
}
