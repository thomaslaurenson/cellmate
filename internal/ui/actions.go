package ui

import (
	"fmt"
	"strconv"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/deal"
	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/game"
	"github.com/thomaslaurenson/cellmate/internal/gfx"
	"github.com/thomaslaurenson/cellmate/internal/stats"
	"github.com/thomaslaurenson/cellmate/internal/store"
)

// handleKey runs the action for a function key, reporting whether the key
// meant anything.
func (t *table) handleKey(in input) bool {
	switch in.key {
	case gfx.KeyF1:
		t.showHelp()
	case gfx.KeyF2:
		t.newGame()
	case gfx.KeyF3:
		t.selectGame()
	case gfx.KeyF4:
		t.showStats()
	case gfx.KeyF5:
		t.showOptions()
	case gfx.KeyF10:
		if in.ctrl && in.shift {
			t.showDebug()
		} else if t.edition.HasUndo() {
			t.undo()
		}
	default:
		return false
	}
	return true
}

// leave finishes with the current game before then runs. A game left after
// at least one move and before it is won counts as a loss, so the player is
// asked first; a game that has already reached a dead end is recorded
// without asking, since the player has been told. The impossible deals
// count for nothing, as in Windows, so there is nothing to ask about.
func (t *table) leave(then func()) {
	switch {
	case t.g.Won(), deal.Impossible(t.g.Number()):
		then()
	case t.over:
		t.recordLoss()
		then()
	case !t.g.CanUndo():
		then()
	default:
		t.dialog = newDialog("cellmate", "Do you want to resign this game?",
			[]string{"Yes", "No"}, func(i int) {
				if i == 0 {
					t.recordLoss()
					then()
				}
			})
	}
}

func (t *table) newGame() {
	t.leave(func() { t.newDeal(t.randomDeal()) })
}

func (t *table) restartGame() {
	n := t.g.Number()
	t.leave(func() { t.newDeal(n) })
}

func (t *table) selectGame() {
	top := t.edition.MaxDeal()
	d := newDialog("Game Number", fmt.Sprintf("Select a game number from 1 to %d:", top),
		[]string{"OK", "Cancel"}, nil)
	d.field = newTextField(strconv.Itoa(t.randomDeal()), len(strconv.Itoa(top)))
	d.choose = func(i int) {
		if i != 0 {
			return
		}
		n, err := strconv.Atoi(d.field.text)
		if err != nil || t.edition.CheckDeal(n) != nil {
			t.dialog = newDialog("cellmate",
				fmt.Sprintf("Please enter a game number from 1 to %d.", top),
				[]string{"OK"}, func(int) { t.selectGame() })
			return
		}
		t.leave(func() { t.newDeal(n) })
	}
	t.dialog = d
}

func (t *table) undo() {
	if t.g.Undo() {
		t.selected = nil
		t.over = false
		t.changed = true
	}
}

func (t *table) exit() { t.quit = true }

// exitGame closes the window, asking first when that would resign a game in
// progress, as leaving any other way does.
func (t *table) exitGame() { t.leave(t.exit) }

// recordWin and recordLoss add the game to the record, unless it is one of
// the impossible deals, which Windows kept out of the statistics.
func (t *table) recordWin() {
	if deal.Impossible(t.g.Number()) {
		return
	}
	t.total.Win()
	t.session.Win()
	t.save()
}

func (t *table) recordLoss() {
	if deal.Impossible(t.g.Number()) {
		return
	}
	t.total.Lose()
	t.session.Lose()
	t.save()
}

// save hands the settings and totals to the store. A failure is reported
// once; after that the game carries on unsaved rather than nagging.
func (t *table) save() {
	if t.store == nil || t.saveFailed {
		return
	}
	d := store.Defaults()
	d.Settings = t.settings
	d.Stats = t.total
	if err := t.store.Save(d); err != nil {
		t.saveFailed = true
		t.dialog = newDialog("cellmate",
			"Your statistics and options could not be saved:\n"+err.Error(),
			[]string{"OK"}, func(int) {})
	}
}

func (t *table) showStats() {
	pct := func(won, lost, p int) string {
		return fmt.Sprintf("won %d, lost %d, %d%%", won, lost, p)
	}
	s, a := t.session, t.total
	msg := fmt.Sprintf("This session: %s\nTotal:        %s\n\nStreaks\n"+
		"  Wins:    %d\n  Losses:  %d\n  Current: %s",
		pct(s.Won, s.Lost, s.Percent()), pct(a.Won, a.Lost, a.Percent()),
		a.BestWinStreak, a.BestLossStreak, a.Current())
	t.dialog = newDialog("Statistics", msg, []string{"OK", "Clear"}, func(i int) {
		if i != 1 {
			return
		}
		t.dialog = newDialog("cellmate", "Clear all statistics?", []string{"Yes", "No"}, func(j int) {
			if j == 0 {
				t.total = stats.Record{}
				t.session = stats.Record{}
				t.save()
			}
			t.showStats()
		})
	})
}

func (t *table) showOptions() {
	d := newDialog("Options", "", []string{"OK", "Cancel"}, nil)
	messages := &checkbox{label: "Display messages on illegal moves", on: t.settings.Messages}
	quick := &checkbox{label: "Quick play (no animation)", on: t.settings.QuickPlay}
	double := &checkbox{label: "Double-click moves card to free cell", on: t.settings.DoubleClick}
	number := &checkbox{label: "Show game number on the table", on: t.settings.GameNumber}
	w95 := &checkbox{label: "Windows 95 edition", on: t.edition == edition.Win95, group: 1}
	wxp := &checkbox{label: "Windows XP edition", on: t.edition == edition.WinXP, group: 1}
	d.checks = []*checkbox{messages, quick, double, number, w95, wxp}
	d.choose = func(i int) {
		if i != 0 {
			return
		}
		t.settings.Messages = messages.on
		t.settings.QuickPlay = quick.on
		t.settings.DoubleClick = double.on
		t.settings.GameNumber = number.on
		t.settings.Edition = t.edition.String()
		t.save()

		ed := edition.WinXP
		if w95.on {
			ed = edition.Win95
		}
		if ed == t.edition {
			return
		}
		// The editions differ in their deal range and in Undo, so a new
		// edition starts a new game rather than changing the rules of one
		// under way, and leaving that game asks first as it always does.
		t.leave(func() {
			t.edition = ed
			t.settings.Edition = ed.String()
			t.save()
			t.newDeal(t.randomDeal())
		})
	}
	t.dialog = d
}

func (t *table) showHelp() {
	t.dialog = newDialog("How to Play",
		"Move every card to the four home cells at the top right,\n"+
			"building each suit up from the ace to the king.\n\n"+
			"In the columns, a card goes on one of the opposite\n"+
			"colour and one rank higher. Any card can go in an empty\n"+
			"free cell at the top left, or in an empty column.\n\n"+
			"Click a card, then click where it should go. Several\n"+
			"cards move together when the free cells allow it.\n"+
			"Double-click a card to send it to a free cell, and\n"+
			"hold the right button on a card to see it in full.",
		[]string{"OK"}, func(int) {})
}

func (t *table) showAbout() {
	title := "cellmate"
	if t.version != "" {
		title += " " + t.version
	}
	t.dialog = newDialog("About cellmate",
		title+"\n\nFreeCell in the style of Windows 95 and XP.\n"+
			"Game numbers match the Microsoft deals.\n\n"+
			"FreeCell was created by Paul Alfille in 1978.\n"+
			"The Windows version was written by Jim Horne.",
		[]string{"OK"}, func(int) {})
}

// showDebug is the hidden box behind Ctrl+Shift+F10, which the Windows game
// kept for testing its ending messages.
func (t *table) showDebug() {
	t.dialog = newDialog("cellmate",
		"Choose Abort to win, Retry to lose,\nor Ignore to cancel.",
		[]string{"Abort", "Retry", "Ignore"}, func(i int) {
			switch i {
			case 0:
				var s game.State
				for i := range game.Slots {
					s.Cells[i] = card.None
					s.Homes[i] = card.New(card.King, card.Suit(i))
				}
				t.g = game.FromState(t.g.Number(), s)
				t.selected, t.over, t.changed = nil, false, true
			case 1:
				t.over = true
				t.dialog = newDialog("Game Over", "Sorry, you lose.\n\nDo you want to play again?",
					[]string{"Yes", "No"}, t.playAgain)
			}
		})
}
