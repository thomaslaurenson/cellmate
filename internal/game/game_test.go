package game

import (
	"errors"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/deal"
)

// setup builds a game from text: cells and homes are four cards or "--",
// and each column is listed top to bottom. Cards not placed anywhere are
// simply absent, which the rules never notice.
func setup(t *testing.T, cells, homes string, cols ...string) *Game {
	t.Helper()
	g := &Game{}
	parse := func(s string) card.Card {
		t.Helper()
		if s == "--" {
			return card.None
		}
		c, err := card.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	for i, s := range strings.Fields(cells) {
		g.board.cells[i] = parse(s)
	}
	for i, s := range strings.Fields(homes) {
		g.board.homes[i] = parse(s)
	}
	for i := range deal.Columns {
		if i < len(cols) {
			for _, s := range strings.Fields(cols[i]) {
				g.board.columns[i] = append(g.board.columns[i], parse(s))
			}
		}
	}
	return g
}

const (
	noCells = "-- -- -- --"
	noHomes = "-- -- -- --"
)

func col(i int) Pos  { return Pos{Area: Column, Index: i} }
func cell(i int) Pos { return Pos{Area: FreeCell, Index: i} }
func home(i int) Pos { return Pos{Area: Home, Index: i} }

func TestNew(t *testing.T) {
	t.Parallel()
	g := New(1)
	if g.Number() != 1 || g.CardsLeft() != card.Count || g.Won() || g.CanUndo() {
		t.Fatalf("fresh game: number %d, left %d, won %v, undo %v",
			g.Number(), g.CardsLeft(), g.Won(), g.CanUndo())
	}
	if got := g.TopAt(col(0)).String(); got != "6S" {
		t.Errorf("column 0 exposes %s, want 6S", got)
	}
	if g.Cell(0) != card.None || g.HomeTop(0) != card.None {
		t.Error("fresh game has cards in the top row")
	}
}

func TestMaxRun(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		cells         string
		emptyCols     int
		toEmpty, want int
	}{
		{name: "all cells free", cells: noCells, want: 5},
		{name: "no cells free", cells: "AS AH AD AC", want: 1},
		{name: "two cells, one empty column", cells: "AS AH -- --", emptyCols: 1, want: 6},
		{name: "four cells, two empty columns", cells: noCells, emptyCols: 2, want: 20},
		{name: "into the only empty column", cells: noCells, emptyCols: 1, toEmpty: 1, want: 5},
		{name: "into one of two empty columns", cells: "AS -- -- --", emptyCols: 2, toEmpty: 1, want: 8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cols := make([]string, deal.Columns)
			for i := range cols {
				if i >= tc.emptyCols {
					cols[i] = "KS"
				}
			}
			g := setup(t, tc.cells, noHomes, cols...)
			if got := g.MaxRun(tc.toEmpty == 1); got != tc.want {
				t.Errorf("MaxRun = %d, want %d", got, tc.want)
			}
		})
	}
}

// full returns the given columns with a king in every other one, so no
// column is empty unless a test says so.
func full(cols ...string) []string {
	out := make([]string, deal.Columns)
	copy(out, cols)
	for i := range out {
		if out[i] == "" {
			out[i] = "KC"
		}
	}
	return out
}

func TestPlay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cells       string
		homes       string
		cols        []string
		move        Move
		want        Move
		wantErr     error
		wantTooMany *TooManyError
	}{
		{
			name: "column to empty free cell",
			cols: full("9H 5S"), move: Move{From: col(0), To: cell(2)},
			want: Move{From: col(0), To: cell(2), Count: 1},
		},
		{
			name: "free cell already full", cells: "2D -- -- --",
			cols: full("9H 5S"), move: Move{From: col(0), To: cell(0)}, wantErr: ErrIllegal,
		},
		{
			name: "ace to any home cell goes to the one clicked",
			cols: full("9H AS"), move: Move{From: col(0), To: home(3)},
			want: Move{From: col(0), To: home(3), Count: 1},
		},
		{
			name: "two finds its ace wherever it is", homes: "-- AS -- --",
			cols: full("9H 2S"), move: Move{From: col(0), To: home(0)},
			want: Move{From: col(0), To: home(1), Count: 1},
		},
		{
			name: "wrong card home", homes: "AS -- -- --",
			cols: full("9H 3S"), move: Move{From: col(0), To: home(0)}, wantErr: ErrIllegal,
		},
		{
			name: "single card onto opposite colour",
			cols: full("4C 9H", "2D TS"), move: Move{From: col(0), To: col(1)},
			want: Move{From: col(0), To: col(1), Count: 1},
		},
		{
			name: "same colour refused",
			cols: full("4C 9S", "2D TS"), move: Move{From: col(0), To: col(1)}, wantErr: ErrIllegal,
		},
		{
			name: "run finds the card that fits",
			cols: full("KD 8C 7H 6S 5H", "3D 9D"), move: Move{From: col(0), To: col(1)},
			want: Move{From: col(0), To: col(1), Count: 4},
		},
		{
			name: "run longer than the cells allow", cells: "AS AH AD --",
			cols: full("KD 8C 7H 6S 5H", "3D 9D"), move: Move{From: col(0), To: col(1)},
			wantTooMany: &TooManyError{Need: 4, Max: 2},
		},
		{
			name:  "free cell to column",
			cells: "8H -- -- --", cols: full("9S"), move: Move{From: cell(0), To: col(0)},
			want: Move{From: cell(0), To: col(0), Count: 1},
		},
		{
			name:  "free cell to column that does not fit",
			cells: "8S -- -- --", cols: full("9S"), move: Move{From: cell(0), To: col(0)}, wantErr: ErrIllegal,
		},
		{
			name:  "run to empty column moves as much as fits",
			cells: "AS AH -- --",
			cols:  []string{"KD 9C 8H 7S 6H 5C", "", "KC", "KC", "KC", "KC", "KC", "KC"},
			move:  Move{From: col(0), To: col(1)}, want: Move{From: col(0), To: col(1), Count: 3},
		},
		{
			name: "single card to empty column on request",
			cols: []string{"KD 9C 8H", "", "KC", "KC", "KC", "KC", "KC", "KC"},
			move: Move{From: col(0), To: col(1), Count: 1}, want: Move{From: col(0), To: col(1), Count: 1},
		},
		{
			name: "requested count beyond the run",
			cols: []string{"KD 9C 8H", "", "KC", "KC", "KC", "KC", "KC", "KC"},
			move: Move{From: col(0), To: col(1), Count: 3}, wantErr: ErrIllegal,
		},
		{
			name: "requested count beyond the limit", cells: "AS AH AD AC",
			cols: []string{"KD 9C 8H", "", "KC", "KC", "KC", "KC", "KC", "KC"},
			move: Move{From: col(0), To: col(1), Count: 2}, wantTooMany: &TooManyError{Need: 2, Max: 1},
		},
		{
			name: "nothing to move", cols: []string{"", "KC"},
			move: Move{From: col(0), To: col(1)}, wantErr: ErrIllegal,
		},
		{
			name: "cards never leave home", homes: "AS -- -- --",
			cols: full(), move: Move{From: home(0), To: cell(0)}, wantErr: ErrIllegal,
		},
		{
			name: "from nowhere", cols: full("9H"),
			move: Move{To: col(1)}, wantErr: ErrIllegal,
		},
		{
			name: "to nowhere", cols: full("9H"),
			move: Move{From: col(0)}, wantErr: ErrIllegal,
		},
		{
			name: "onto itself", cols: full("9H"),
			move: Move{From: col(0), To: col(0)}, wantErr: ErrIllegal,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cells, homes := tc.cells, tc.homes
			if cells == "" {
				cells = noCells
			}
			if homes == "" {
				homes = noHomes
			}
			g := setup(t, cells, homes, tc.cols...)
			before := g.CardsLeft()
			got, err := g.Play(tc.move)
			switch {
			case tc.wantTooMany != nil:
				var tm *TooManyError
				if !errors.As(err, &tm) || *tm != *tc.wantTooMany {
					t.Fatalf("error = %v, want %v", err, tc.wantTooMany)
				}
			case tc.wantErr != nil:
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %v, want %v", err, tc.wantErr)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tc.want {
					t.Errorf("Play = %+v, want %+v", got, tc.want)
				}
				if !g.CanUndo() {
					t.Error("move was not recorded for undo")
				}
				return
			}
			if g.CanUndo() || g.CardsLeft() != before {
				t.Error("a refused move changed the game")
			}
		})
	}
}

func TestPlayMovesCards(t *testing.T) {
	t.Parallel()
	g := setup(t, noCells, noHomes, full("KD 8C 7H 6S 5H", "3D 9D")...)
	if _, err := g.Play(Move{From: col(0), To: col(1)}); err != nil {
		t.Fatal(err)
	}
	if got := len(g.ColumnCards(0)); got != 1 {
		t.Errorf("source column has %d cards, want 1", got)
	}
	var names []string
	for _, c := range g.ColumnCards(1) {
		names = append(names, c.String())
	}
	if got := strings.Join(names, " "); got != "3D 9D 8C 7H 6S 5H" {
		t.Errorf("destination column = %s", got)
	}
}

func TestCheckLeavesGameAlone(t *testing.T) {
	t.Parallel()
	g := setup(t, noCells, noHomes, full("9H 5S")...)
	if _, err := g.Check(Move{From: col(0), To: cell(0)}); err != nil {
		t.Fatal(err)
	}
	if g.Cell(0) != card.None || g.CanUndo() {
		t.Error("Check changed the game")
	}
}

func TestToFreeCell(t *testing.T) {
	t.Parallel()
	g := setup(t, "2D -- -- --", noHomes, full("9H 5S")...)
	m, err := g.ToFreeCell(col(0))
	if err != nil {
		t.Fatal(err)
	}
	if m.To != cell(1) {
		t.Errorf("went to %+v, want the first empty cell", m.To)
	}

	g = setup(t, "2D 3D 4D 5D", noHomes, full("9H 5S")...)
	if _, err := g.ToFreeCell(col(0)); !errors.Is(err, ErrIllegal) {
		t.Errorf("with every cell full, error = %v", err)
	}
}

func TestAutoStep(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		cells string
		homes string
		cols  []string
		want  []string // cards sent home, in order
	}{
		{name: "nothing safe", homes: noHomes, cols: full("5H"), want: nil},
		{name: "aces and twos always go", homes: noHomes, cols: full("2C AC", "AH"), want: []string{"AC", "2C", "AH"}},
		{name: "a three waits for both opposite twos", homes: "2C AS AH --", cols: full("3H 2H"), want: []string{"2H"}},
		{name: "a three goes once both are home", homes: "2C 2S 2H --", cols: full("3H"), want: []string{"3H"}},
		{name: "free cells are checked first", cells: "AD -- -- --", homes: noHomes, cols: full("AS"), want: []string{"AD", "AS"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cells := tc.cells
			if cells == "" {
				cells = noCells
			}
			g := setup(t, cells, tc.homes, tc.cols...)
			var got []string
			for {
				m, ok := g.AutoStep()
				if !ok {
					break
				}
				if m.To.Area != Home || m.Count != 1 {
					t.Fatalf("automatic move %+v is not a single card home", m)
				}
				got = append(got, g.HomeTop(m.To.Index).String())
			}
			if strings.Join(got, " ") != strings.Join(tc.want, " ") {
				t.Errorf("went home: %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUndoTakesBackAutomaticMoves(t *testing.T) {
	t.Parallel()
	g := setup(t, noCells, noHomes, full("AC 5H", "6S")...)
	if _, err := g.Play(Move{From: col(0), To: col(1)}); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.AutoStep(); !ok {
		t.Fatal("the exposed ace did not go home")
	}
	if !g.Undo() {
		t.Fatal("Undo reported nothing to undo")
	}
	if got := g.TopAt(col(0)).String(); got != "5H" {
		t.Errorf("after undo column 0 exposes %s, want 5H", got)
	}
	if g.HomeTop(0) != card.None {
		t.Error("the ace is still home after undo")
	}
	if g.Undo() {
		t.Error("second Undo found a move to take back")
	}
}

func TestWon(t *testing.T) {
	t.Parallel()
	g := setup(t, noCells, "KC KD KH KS")
	if !g.Won() || g.CardsLeft() != 0 {
		t.Errorf("all kings home: won %v, left %d", g.Won(), g.CardsLeft())
	}
}

func TestHasMoves(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		cells string
		cols  []string
		want  bool
	}{
		{name: "an empty free cell is always a move", cells: "AS AH AD --", cols: full("5H"), want: true},
		{name: "free cell to column", cells: "4S 2H 2D 2C", cols: full("5H"), want: true},
		{name: "column to column", cells: "4H 2H 2D 2C", cols: full("5H 3D", "4C"), want: true},
		{name: "empty column", cells: "4H 2H 2D 2C", cols: []string{"5H", ""}, want: true},
		{name: "stuck", cells: "4H 3H 2D 7C", cols: full("5H", "9H", "TD"), want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := setup(t, tc.cells, "AS AC -- --", tc.cols...)
			if got := g.HasMoves(); got != tc.want {
				t.Errorf("HasMoves = %v, want %v", got, tc.want)
			}
		})
	}
}

// census counts every card on the board, so a test can prove none are lost
// or duplicated.
func census(g *Game) map[card.Card]int {
	seen := map[card.Card]int{}
	for i := range Slots {
		if c := g.Cell(i); c != card.None {
			seen[c]++
		}
		if top := g.HomeTop(i); top != card.None {
			for r := card.Ace; r <= top.Rank(); r++ {
				seen[card.New(r, top.Suit())]++
			}
		}
	}
	for i := range deal.Columns {
		for _, c := range g.ColumnCards(i) {
			seen[c]++
		}
	}
	return seen
}

// TestRandomPlayKeepsTheDeck plays random legal moves across many deals,
// with automatic moves and undo mixed in, and checks after every step that
// all 52 cards are still on the board exactly once.
func TestRandomPlayKeepsTheDeck(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(1, 2))
	areas := []Area{FreeCell, Home, Column}
	for n := 1; n <= 200; n++ {
		g := New(n)
		for step := 0; step < 300; step++ {
			if rng.IntN(20) == 0 {
				g.Undo()
			} else {
				from := Pos{Area: areas[rng.IntN(3)], Index: rng.IntN(deal.Columns)}
				to := Pos{Area: areas[rng.IntN(3)], Index: rng.IntN(deal.Columns)}
				if from.Area != Column {
					from.Index %= Slots
				}
				if to.Area != Column {
					to.Index %= Slots
				}
				if _, err := g.Play(Move{From: from, To: to}); err == nil {
					for {
						if _, ok := g.AutoStep(); !ok {
							break
						}
					}
				}
			}
			seen := census(g)
			if len(seen) != card.Count {
				t.Fatalf("deal %d step %d: %d distinct cards on the board", n, step, len(seen))
			}
			for c, k := range seen {
				if k != 1 {
					t.Fatalf("deal %d step %d: %v appears %d times", n, step, c, k)
				}
			}
		}
	}
}

func TestStateRoundTrip(t *testing.T) {
	t.Parallel()
	g := New(617)
	if _, err := g.Play(Move{From: col(0), To: cell(1)}); err != nil {
		t.Fatal(err)
	}
	s := g.State()
	s.Columns[1] = append(s.Columns[1], card.None) // Changing the copy must not reach the game
	r := FromState(617, g.State())
	if r.Number() != 617 || r.CanUndo() {
		t.Errorf("restored game: number %d, undo %v", r.Number(), r.CanUndo())
	}
	if r.Cell(1) != g.Cell(1) || len(r.ColumnCards(0)) != len(g.ColumnCards(0)) {
		t.Error("restored board differs")
	}
	if len(g.ColumnCards(1)) != 7 {
		t.Error("editing a State changed the game it came from")
	}
}
