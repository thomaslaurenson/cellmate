// Package game holds the rules of FreeCell as the Microsoft versions play
// them: which moves are legal, how many cards can move at once, which cards
// go home on their own, and when the game is won or stuck.
//
// The package knows nothing about drawing or input. The ui package asks it
// what a click means and animates what it reports.
package game

import (
	"errors"
	"fmt"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/deal"
)

// Slots is the number of free cells, and also the number of home cells.
const Slots = 4

// Area names a part of the board.
type Area int

// The board areas.
const (
	FreeCell Area = iota + 1
	Home
	Column
)

// Pos is a place on the board: a free cell, a home cell or a column, with
// Index counting from the left.
type Pos struct {
	Area  Area
	Index int
}

// Move is one move as made or reported by the game. Count is the number of
// cards moved; for a move asked of Play, zero means as many as fit.
type Move struct {
	From, To Pos
	Count    int
}

// ErrIllegal is returned for a move the rules do not allow.
var ErrIllegal = errors.New("that move is not allowed")

// TooManyError is returned when a run could go to its destination but is
// longer than the free cells and empty columns allow.
type TooManyError struct {
	Need, Max int
}

func (e *TooManyError) Error() string {
	return fmt.Sprintf("run of %d cards is over the limit of %d", e.Need, e.Max)
}

// board is the part of the game that undo restores.
type board struct {
	cells   [Slots]card.Card
	homes   [Slots]card.Card
	columns [deal.Columns][]card.Card
}

func (b board) clone() board {
	c := b
	for i, col := range b.columns {
		c.columns[i] = append([]card.Card(nil), col...)
	}
	return c
}

// Game is one deal in progress.
type Game struct {
	number  int
	board   board
	history []board
}

// New deals game n.
func New(n int) *Game {
	g := &Game{number: n}
	for i := range Slots {
		g.board.cells[i] = card.None
		g.board.homes[i] = card.None
	}
	g.board.columns = deal.Layout(n)
	return g
}

// State is a snapshot of the board, which FromState builds a game from.
// Empty free cells and home cells hold card.None.
type State struct {
	Cells   [Slots]card.Card
	Homes   [Slots]card.Card
	Columns [deal.Columns][]card.Card
}

// State returns a copy of the board.
func (g *Game) State() State {
	b := g.board.clone()
	return State{Cells: b.cells, Homes: b.homes, Columns: b.columns}
}

// FromState rebuilds game n from a snapshot, with no moves to undo.
func FromState(n int, s State) *Game {
	b := board{cells: s.Cells, homes: s.Homes, columns: s.Columns}
	return &Game{number: n, board: b.clone()}
}

// Number returns the deal number.
func (g *Game) Number() int { return g.number }

// Cell returns the card in free cell i, or card.None.
func (g *Game) Cell(i int) card.Card { return g.board.cells[i] }

// HomeTop returns the top card of home cell i, or card.None.
func (g *Game) HomeTop(i int) card.Card { return g.board.homes[i] }

// ColumnCards returns column i from the top of the pile to the exposed card.
// The slice belongs to the game and must not be changed.
func (g *Game) ColumnCards(i int) []card.Card { return g.board.columns[i] }

// CardsLeft returns the number of cards not yet home.
func (g *Game) CardsLeft() int {
	n := card.Count
	for _, top := range g.board.homes {
		if top != card.None {
			n -= int(top.Rank())
		}
	}
	return n
}

// Won reports whether every card is home.
func (g *Game) Won() bool { return g.CardsLeft() == 0 }

// CanUndo reports whether there is a move to take back.
func (g *Game) CanUndo() bool { return len(g.history) > 0 }

// Undo takes back the last move made through Play, along with any cards that
// went home automatically after it.
func (g *Game) Undo() bool {
	if len(g.history) == 0 {
		return false
	}
	g.board = g.history[len(g.history)-1]
	g.history = g.history[:len(g.history)-1]
	return true
}

// TopAt returns the exposed card at p, or card.None if p is empty.
func (g *Game) TopAt(p Pos) card.Card {
	switch p.Area {
	case FreeCell:
		return g.board.cells[p.Index]
	case Home:
		return g.board.homes[p.Index]
	case Column:
		col := g.board.columns[p.Index]
		if len(col) == 0 {
			return card.None
		}
		return col[len(col)-1]
	}
	return card.None
}

// emptyCells and emptyColumns count what a multi-card move can use.
func (g *Game) emptyCells() int {
	n := 0
	for _, c := range g.board.cells {
		if c == card.None {
			n++
		}
	}
	return n
}

func (g *Game) emptyColumns() int {
	n := 0
	for _, col := range g.board.columns {
		if len(col) == 0 {
			n++
		}
	}
	return n
}

// MaxRun returns how many cards can move at once. Each free cell holds one
// card, and each empty column doubles what can be shuffled through: the
// formula is (free cells + 1) * 2^(empty columns). A move into an empty
// column cannot use that column as a staging place, so it is left out.
func (g *Game) MaxRun(toEmptyColumn bool) int {
	cols := g.emptyColumns()
	if toEmptyColumn {
		cols--
	}
	return (g.emptyCells() + 1) << max(cols, 0)
}

// stacks reports whether lower can sit on upper in a column: one rank lower
// and the opposite colour.
func stacks(lower, upper card.Card) bool {
	return lower.Rank()+1 == upper.Rank() && lower.Red() != upper.Red()
}

// runLength returns how many cards at the bottom of a non-empty column form
// a sequence that could move together.
func runLength(col []card.Card) int {
	n := 1
	for i := len(col) - 1; i > 0 && stacks(col[i], col[i-1]); i-- {
		n++
	}
	return n
}

// homeFor returns the home cell that would take c, preferring prefer when it
// can. It returns -1 when no home cell takes c.
func (g *Game) homeFor(c card.Card, prefer int) int {
	fits := func(i int) bool {
		top := g.board.homes[i]
		if top == card.None {
			return c.Rank() == card.Ace
		}
		return top.Suit() == c.Suit() && top.Rank()+1 == c.Rank()
	}
	if prefer >= 0 && prefer < Slots && fits(prefer) {
		return prefer
	}
	for i := range Slots {
		if fits(i) {
			return i
		}
	}
	return -1
}

// resolve turns a requested move into the exact move the rules allow,
// without changing the board.
func (g *Game) resolve(m Move) (Move, error) {
	src := g.TopAt(m.From)
	if src == card.None || m.From.Area == Home || m.From == m.To {
		return Move{}, ErrIllegal
	}
	switch m.To.Area {
	case FreeCell:
		if g.board.cells[m.To.Index] != card.None {
			return Move{}, ErrIllegal
		}
		return Move{From: m.From, To: m.To, Count: 1}, nil
	case Home:
		i := g.homeFor(src, m.To.Index)
		if i < 0 {
			return Move{}, ErrIllegal
		}
		return Move{From: m.From, To: Pos{Area: Home, Index: i}, Count: 1}, nil
	case Column:
		return g.resolveToColumn(m)
	}
	return Move{}, ErrIllegal
}

func (g *Game) resolveToColumn(m Move) (Move, error) {
	dst := g.board.columns[m.To.Index]
	if m.From.Area == FreeCell {
		c := g.board.cells[m.From.Index]
		if len(dst) > 0 && !stacks(c, dst[len(dst)-1]) {
			return Move{}, ErrIllegal
		}
		return Move{From: m.From, To: m.To, Count: 1}, nil
	}

	src := g.board.columns[m.From.Index]
	run := runLength(src)
	toEmpty := len(dst) == 0
	limit := g.MaxRun(toEmpty)

	if toEmpty {
		n := min(run, limit)
		if m.Count > 0 {
			if m.Count > run {
				return Move{}, ErrIllegal
			}
			if m.Count > limit {
				return Move{}, &TooManyError{Need: m.Count, Max: limit}
			}
			n = m.Count
		}
		return Move{From: m.From, To: m.To, Count: n}, nil
	}

	// Only one length of run can land on a given card, so find it
	top := dst[len(dst)-1]
	for n := 1; n <= run; n++ {
		if stacks(src[len(src)-n], top) {
			if n > limit {
				return Move{}, &TooManyError{Need: n, Max: limit}
			}
			return Move{From: m.From, To: m.To, Count: n}, nil
		}
	}
	return Move{}, ErrIllegal
}

// apply carries out a resolved move.
func (g *Game) apply(m Move) {
	b := &g.board
	var moving []card.Card
	switch m.From.Area {
	case FreeCell:
		moving = []card.Card{b.cells[m.From.Index]}
		b.cells[m.From.Index] = card.None
	case Column:
		col := b.columns[m.From.Index]
		moving = append([]card.Card(nil), col[len(col)-m.Count:]...)
		b.columns[m.From.Index] = col[:len(col)-m.Count]
	}
	switch m.To.Area {
	case FreeCell:
		b.cells[m.To.Index] = moving[0]
	case Home:
		b.homes[m.To.Index] = moving[0]
	case Column:
		b.columns[m.To.Index] = append(b.columns[m.To.Index], moving...)
	}
}

// Check reports the move Play would make for m, without making it.
func (g *Game) Check(m Move) (Move, error) {
	return g.resolve(m)
}

// Play makes a move and records it for Undo. For a move into a column Count
// may be zero, meaning as many cards as the rules allow; a positive Count
// only matters for a move into an empty column, where it chooses between
// moving the whole run and a single card.
func (g *Game) Play(m Move) (Move, error) {
	r, err := g.resolve(m)
	if err != nil {
		return Move{}, err
	}
	g.history = append(g.history, g.board.clone())
	g.apply(r)
	return r, nil
}

// ToFreeCell moves the exposed card at from into the first empty free cell,
// as a double-click does.
func (g *Game) ToFreeCell(from Pos) (Move, error) {
	for i, c := range g.board.cells {
		if c == card.None {
			return g.Play(Move{From: from, To: Pos{Area: FreeCell, Index: i}})
		}
	}
	return Move{}, ErrIllegal
}

// safeHome reports whether c can go home without the player ever wanting it
// back: aces and twos always can, and any other card once both cards of the
// opposite colour one rank lower are home, since nothing could then be
// placed on it in a column.
func (g *Game) safeHome(c card.Card) bool {
	if c.Rank() <= 2 {
		return true
	}
	need := 0
	for _, top := range g.board.homes {
		if top != card.None && top.Red() != c.Red() && top.Rank() >= c.Rank()-1 {
			need++
		}
	}
	return need == 2
}

// AutoStep sends one card home if any exposed card can safely go, and
// reports the move. The ui calls it repeatedly, animating each move, until it
// reports false. Automatic moves are not recorded separately for Undo: they
// are taken back with the move that caused them.
func (g *Game) AutoStep() (Move, bool) {
	try := func(from Pos) (Move, bool) {
		c := g.TopAt(from)
		if c == card.None || !g.safeHome(c) {
			return Move{}, false
		}
		i := g.homeFor(c, -1)
		if i < 0 {
			return Move{}, false
		}
		m := Move{From: from, To: Pos{Area: Home, Index: i}, Count: 1}
		g.apply(m)
		return m, true
	}
	for i := range Slots {
		if m, ok := try(Pos{Area: FreeCell, Index: i}); ok {
			return m, true
		}
	}
	for i := range deal.Columns {
		if m, ok := try(Pos{Area: Column, Index: i}); ok {
			return m, true
		}
	}
	return Move{}, false
}

// HasMoves reports whether any legal move remains. A game with an empty free
// cell and a card to put in it always has one, so a stuck game is one with
// every free cell full.
func (g *Game) HasMoves() bool {
	var froms []Pos
	for i := range Slots {
		froms = append(froms, Pos{Area: FreeCell, Index: i})
	}
	for i := range deal.Columns {
		froms = append(froms, Pos{Area: Column, Index: i})
	}
	var tos []Pos
	for i := range Slots {
		tos = append(tos, Pos{Area: FreeCell, Index: i}, Pos{Area: Home, Index: i})
	}
	for i := range deal.Columns {
		tos = append(tos, Pos{Area: Column, Index: i})
	}
	for _, f := range froms {
		for _, t := range tos {
			if f.Area == FreeCell && t.Area == FreeCell {
				continue
			}
			if _, err := g.resolve(Move{From: f, To: t, Count: 1}); err == nil {
				return true
			}
		}
	}
	return false
}
