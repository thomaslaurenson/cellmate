package deal

import (
	"strings"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/card"
)

// The published deals from the Rosetta Code task "Deal cards for FreeCell",
// which match the Windows game.
func TestCards(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		deal int
		want string
	}{
		{
			name: "deal 1",
			deal: 1,
			want: "JD 2D 9H JC 5D 7H 7C 5H KD KC 9S 5S AD QC KH 3H 2S KS 9D QD JS AS AH 3C " +
				"4C 5C TS QH 4H AC 4D 7S 3S TD 4S TH 8H 2C JH 7D 6D 8S 8D QS 6C 3D 8C TC " +
				"6S 9C 2H 6H",
		},
		{
			name: "deal 617",
			deal: 617,
			want: "7D AD 5C 3S 5S 8C 2D AH TD 7S QD AC 6D 8H AS KH TH QC 3H 9D 6S 8D 3D TC " +
				"KD 5H 9S 3C 8S 7H 4D JS 4C QS 9C 9H 7C 6H 2C 2S 4S TS 2H 5D JC 6C JH QH " +
				"JD KS KC 4H",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cards := Cards(tc.deal)
			got := make([]string, len(cards))
			for i, c := range cards {
				got[i] = c.String()
			}
			if s := strings.Join(got, " "); s != tc.want {
				t.Errorf("Cards(%d) =\n%s\nwant\n%s", tc.deal, s, tc.want)
			}
		})
	}
}

func TestCardsIsPermutation(t *testing.T) {
	t.Parallel()
	for _, n := range []int{-2, -1, 1, 11982, 32000, 32001, 999999, 1000000} {
		var seen [card.Count]bool
		for _, c := range Cards(n) {
			if !c.Valid() || seen[c] {
				t.Fatalf("Cards(%d) repeats or invents card %v", n, c)
			}
			seen[c] = true
		}
	}
}

func TestLayout(t *testing.T) {
	t.Parallel()
	cols := Layout(1)
	for i, col := range cols {
		want := 6
		if i < 4 {
			want = 7
		}
		if len(col) != want {
			t.Errorf("column %d has %d cards, want %d", i, len(col), want)
		}
	}
	if got := cols[0][0].String(); got != "JD" {
		t.Errorf("column 0 starts with %s, want JD", got)
	}
	if got := cols[7][5].String(); got != "TC" {
		t.Errorf("column 7 ends with %s, want TC", got)
	}
}

func TestWriteRows(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	if err := WriteRows(&b, 1); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
	if len(lines) != 7 {
		t.Fatalf("got %d rows, want 7:\n%s", len(lines), b.String())
	}
	if lines[0] != "JD 2D 9H JC 5D 7H 7C 5H" {
		t.Errorf("first row = %q", lines[0])
	}
	if lines[6] != "6S 9C 2H 6H" {
		t.Errorf("last row = %q", lines[6])
	}
}

// TestImpossibleDeals pins the two hand-made deals to what the Windows game
// shows for them, row by row.
func TestImpossibleDeals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		deal int
		rows []string
	}{
		{
			deal: -1,
			rows: []string{
				"AC AD AH AS QC QD QH QS",
				"3C 3D 3H 3S TC TD TH TS",
				"5C 5D 5H 5S 8C 8D 8H 8S",
				"7C 7D 7H 7S 6C 6D 6H 6S",
				"9C 9D 9H 9S 4C 4D 4H 4S",
				"JC JD JH JS 2C 2D 2H 2S",
				"KC KD KH KS",
			},
		},
		{
			deal: -2,
			rows: []string{
				"AS AH AD AC 7S 7H 7D 7C",
				"KS KH KD KC 6S 6H 6D 6C",
				"QS QH QD QC 5S 5H 5D 5C",
				"JS JH JD JC 4S 4H 4D 4C",
				"TS TH TD TC 3S 3H 3D 3C",
				"9S 9H 9D 9C 2S 2H 2D 2C",
				"8S 8H 8D 8C",
			},
		},
	}
	for _, tc := range tests {
		var b strings.Builder
		if err := WriteRows(&b, tc.deal); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSuffix(b.String(), "\n"); got != strings.Join(tc.rows, "\n") {
			t.Errorf("deal %d =\n%s\nwant\n%s", tc.deal, got, strings.Join(tc.rows, "\n"))
		}
	}
}

// TestImpossibleDealsBuryTheAces checks the property that makes the two
// deals impossible: every ace is at the top of one of the first four
// columns, under six cards of its own suit, none of which can be built on
// another card of the column.
func TestImpossibleDealsBuryTheAces(t *testing.T) {
	t.Parallel()
	for _, n := range []int{-1, -2} {
		cols := Layout(n)
		for i := range 4 {
			col := cols[i]
			if len(col) != 7 || col[0].Rank() != card.Ace {
				t.Errorf("deal %d column %d = %v, want an ace under six cards", n, i, col)
				continue
			}
			for _, c := range col[1:] {
				if c.Suit() != col[0].Suit() {
					t.Errorf("deal %d column %d holds %v with the %v", n, i, c, col[0])
				}
			}
		}
	}
}

func TestImpossible(t *testing.T) {
	t.Parallel()
	for n, want := range map[int]bool{-2: true, -1: true, 0: false, 1: false, -3: false} {
		if got := Impossible(n); got != want {
			t.Errorf("Impossible(%d) = %v, want %v", n, got, want)
		}
	}
}
