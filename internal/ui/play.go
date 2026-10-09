package ui

import (
	"errors"
	"fmt"
	"image"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/game"
)

// step advances the table by one tick. Input goes to the first thing that
// wants it: an open dialog, then a card in flight (which holds everything
// up), then the menus, then the function keys, then the board.
func (t *table) step(in input) {
	t.tick++
	t.cursor = in.cursor

	// Closing the window is leaving the game, so it asks the same question
	// Exit does. With a message box open it is ignored, as Windows ignored
	// its main window while a modal box was up.
	if in.close && t.dialog == nil {
		t.exitGame()
		return
	}

	if d := t.dialog; d != nil {
		if d.update(in) && t.dialog == d {
			t.dialog = nil
		}
		return
	}

	if t.winTick > 0 {
		t.winTick++
		if t.winTick > winTicks {
			t.winTick = 0
			t.showWin()
		}
		return
	}

	if f := t.flight; f != nil {
		f.tick++
		if f.tick >= flightTicks {
			t.flight = nil
			t.autoplay()
		}
		return
	}

	t.reveal = nil
	if t.updateMenus(in) {
		return
	}
	if !t.handleKey(in) {
		switch {
		case in.escape:
			t.selected = nil
		case in.leftPressed:
			t.click(in.cursor)
		case in.rightDown:
			t.reveal = t.revealAt(in.cursor)
		}
		for _, r := range in.chars {
			if t.dialog == nil && t.flight == nil {
				t.typeKey(r)
			}
		}
	}

	if t.flight == nil && t.dialog == nil && t.changed {
		t.changed = false
		t.checkEnd()
	}
}

// click handles a left click: the first click on a card selects it, the
// second click on somewhere else moves it there, and a second click on the
// same column soon after the first sends its card to a free cell.
func (t *table) click(pt image.Point) {
	pos, ok := hitTest(pt)
	if !ok {
		t.selected = nil
		return
	}

	if t.selected == nil {
		if pos.Area != game.Home && t.g.TopAt(pos) != card.None {
			t.selected = &pos
			t.selectTick = t.tick
		}
		return
	}

	from := *t.selected
	t.selected = nil
	if from == pos {
		if pos.Area == game.Column && t.settings.DoubleClick && t.tick-t.selectTick <= doubleClickTicks {
			if _, err := t.g.ToFreeCell(pos); err == nil {
				t.afterMove()
			}
		}
		return
	}
	t.tryMove(from, pos)
}

// tryMove moves from one place to another, first asking whether to move the
// whole run or a single card when the destination is an empty column.
func (t *table) tryMove(from, to game.Pos) {
	m := game.Move{From: from, To: to}
	if to.Area == game.Column && t.g.TopAt(to) == card.None && from.Area == game.Column {
		if r, err := t.g.Check(m); err == nil && r.Count > 1 {
			d := newDialog("Move to Empty Column", "Move the whole run, or just the exposed card?",
				[]string{"Move column", "Move single card", "Cancel"},
				func(i int) {
					switch i {
					case 0:
						t.play(game.Move{From: from, To: to, Count: r.Count})
					case 1:
						t.play(game.Move{From: from, To: to, Count: 1})
					}
				})
			d.hotkeys = []rune{'c', 's', 0}
			t.dialog = d
			return
		}
	}
	t.play(m)
}

func (t *table) play(m game.Move) {
	if _, err := t.g.Play(m); err != nil {
		t.showError(err)
		return
	}
	t.afterMove()
}

func (t *table) showError(err error) {
	if !t.settings.Messages {
		return
	}
	msg := "That move is not allowed."
	var tm *game.TooManyError
	if errors.As(err, &tm) {
		msg = fmt.Sprintf("That move needs %d cards to move at once, but only %d can.", tm.Need, tm.Max)
	}
	t.dialog = newDialog("cellmate", msg, []string{"OK"}, func(int) {})
}

func (t *table) afterMove() {
	t.changed = true
	t.autoplay()
}

// autoplay starts the next safe card flying home, if there is one. With
// quick play on, every safe card goes home at once instead.
func (t *table) autoplay() {
	if t.settings.QuickPlay {
		for {
			if _, ok := t.g.AutoStep(); !ok {
				return
			}
			t.changed = true
		}
	}
	m, ok := t.g.AutoStep()
	if !ok {
		return
	}
	t.changed = true
	from := cellRect(0).Min
	switch m.From.Area {
	case game.FreeCell:
		from = cellRect(m.From.Index).Min
	case game.Column:
		n := len(t.g.ColumnCards(m.From.Index)) + 1
		from = cardPoint(m.From.Index, n-1, n)
	}
	t.flight = &flight{move: m, from: from, to: slotPoint(m.To, 0)}
}

// checkEnd reports a win, or a game with no legal moves left.
func (t *table) checkEnd() {
	if t.over {
		return
	}
	switch {
	case t.g.Won():
		t.over = true
		t.recordWin()
		if t.settings.QuickPlay || t.dialog != nil {
			t.showWin()
			return
		}
		t.winTick = 1
	case !t.g.HasMoves():
		t.over = true
		msg := "There are no more legal moves.\n\nDo you want to play again?"
		if t.edition.HasUndo() {
			msg = "There are no more legal moves.\nPress F10 to undo.\n\nDo you want to play again?"
		}
		t.dialog = newDialog("Game Over", msg, []string{"Yes", "No"}, t.playAgain)
	}
}

// showWin asks whether to play again after a win. A save error raised by
// recording the win is shown first, and this question follows it.
func (t *table) showWin() {
	ask := func() {
		t.dialog = newDialog("Game Over", "Congratulations, you win!\n\nDo you want to play again?",
			[]string{"Yes", "No"}, t.playAgain)
	}
	if d := t.dialog; d != nil {
		choose := d.choose
		d.choose = func(i int) { choose(i); ask() }
		return
	}
	ask()
}

func (t *table) playAgain(i int) {
	if i == 0 {
		t.leave(func() { t.newDeal(t.randomDeal()) })
	}
}

// revealAt returns the buried card under pt, or nil when pt is not over one.
func (t *table) revealAt(pt image.Point) *reveal {
	pos, ok := hitTest(pt)
	if !ok || pos.Area != game.Column {
		return nil
	}
	n := len(t.g.ColumnCards(pos.Index))
	j := cardIndexAt(pt.Y, n)
	if j < 0 || j == n-1 {
		return nil
	}
	return &reveal{column: pos.Index, index: j}
}

// typeKey plays by keyboard, as Windows FreeCell allowed: the digits 1 to 8
// name the columns, 0 the free cells and 9 the home cells. The first key
// picks up a card and the second says where it goes. Pressing 0 again while
// a free cell is picked moves on to the next occupied one.
func (t *table) typeKey(r rune) {
	if r < '0' || r > '9' {
		return
	}
	if t.selected == nil {
		switch {
		case r == '0':
			t.selectNextCell(-1)
		case r != '9':
			pos := game.Pos{Area: game.Column, Index: int(r - '1')}
			if t.g.TopAt(pos) != card.None {
				t.selected = &pos
				t.selectTick = t.tick
			}
		}
		return
	}

	from := *t.selected
	switch {
	case r == '0' && from.Area == game.FreeCell:
		t.selectNextCell(from.Index)
	case r == '0':
		t.selected = nil
		if _, err := t.g.ToFreeCell(from); err != nil {
			t.showError(err)
			return
		}
		t.afterMove()
	case r == '9':
		t.selected = nil
		t.play(game.Move{From: from, To: game.Pos{Area: game.Home}})
	default:
		to := game.Pos{Area: game.Column, Index: int(r - '1')}
		t.selected = nil
		if to != from {
			t.tryMove(from, to)
		}
	}
}

// selectNextCell selects the first occupied free cell after index after,
// wrapping round, or clears the selection when every cell is empty.
func (t *table) selectNextCell(after int) {
	for k := 1; k <= game.Slots; k++ {
		i := (after + k + game.Slots) % game.Slots
		if t.g.Cell(i) != card.None {
			pos := game.Pos{Area: game.FreeCell, Index: i}
			t.selected = &pos
			return
		}
	}
	t.selected = nil
}
