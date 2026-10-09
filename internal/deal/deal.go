// Package deal reproduces the numbered deals of Microsoft FreeCell.
//
// The algorithm is the one Jim Horne published: seed the Microsoft C runtime
// rand() with the deal number, then repeatedly pick a card from the remaining
// deck, swap it with the last one, and deal it. Cards are dealt row by row
// across the eight columns, so the first four columns get seven cards and the
// last four get six.
//
// Two deals are not shuffled at all. The game's help file claimed every deal
// could be won, and Horne hid two that cannot at -1 and -2 as the joke behind
// the claim. Both are dealt here as they were in Windows.
package deal

import (
	"io"
	"strings"

	"github.com/thomaslaurenson/cellmate/internal/card"
)

// Columns is the number of tableau columns.
const Columns = 8

// msRand is the linear congruential generator from the Microsoft C runtime.
type msRand struct{ state uint32 }

// next returns the following value in the range 0 to 32767.
//
// Masking the state to 31 bits is what keeps the result to 15 bits. Keeping
// the full 32-bit product and shifting it would leak bit 31 into bit 15 of
// the result, and change every deal.
func (r *msRand) next() int {
	r.state = (r.state*214013 + 2531011) & 0x7fffffff
	return int(r.state >> 16)
}

// fixed is a deal laid out by hand rather than shuffled: the ranks down the
// first four columns, the ranks down the last four, and the suit of each
// column in turn, since every column of the two is one suit.
type fixed struct {
	left, right []card.Rank
	suits       [4]card.Suit
}

// impossible holds the two deals that cannot be won, by number. Each buries
// the four aces at the top of the first four columns under six cards of
// their own suit, so none can be dug out before the free cells fill. Windows
// 95 and XP both dealt them, and neither counted them in the statistics.
//
// They were read off the Windows game: in -1 the odd ranks climb from the
// aces to the kings down the first four columns and the even ranks fall from
// the queens to the twos down the last four; in -2 the kings sit on the aces
// and every column runs down from there.
var impossible = map[int]fixed{
	-1: {
		left:  []card.Rank{card.Ace, 3, 5, 7, 9, card.Jack, card.King},
		right: []card.Rank{card.Queen, 10, 8, 6, 4, 2},
		suits: [4]card.Suit{card.Clubs, card.Diamonds, card.Hearts, card.Spades},
	},
	-2: {
		left:  []card.Rank{card.Ace, card.King, card.Queen, card.Jack, 10, 9, 8},
		right: []card.Rank{7, 6, 5, 4, 3, 2},
		suits: [4]card.Suit{card.Spades, card.Hearts, card.Diamonds, card.Clubs},
	},
}

// Impossible reports whether n is one of the two deals that cannot be won.
func Impossible(n int) bool {
	_, ok := impossible[n]
	return ok
}

// cards lays the deal out in the order the game deals, row by row: a card
// for each of the first four columns, then one for each of the last four,
// until the last four are full and the seventh row is the first four's
// alone.
func (d fixed) cards() [card.Count]card.Card {
	var out [card.Count]card.Card
	i := 0
	for row, r := range d.left {
		for _, s := range d.suits {
			out[i] = card.New(r, s)
			i++
		}
		if row < len(d.right) {
			for _, s := range d.suits {
				out[i] = card.New(d.right[row], s)
				i++
			}
		}
	}
	return out
}

// Cards returns the 52 cards of deal n in the order they are dealt. The
// numbered deals are shuffled from n as their seed; -1 and -2 are the two
// impossible deals, laid out as they were in Windows.
func Cards(n int) [card.Count]card.Card {
	if d, ok := impossible[n]; ok {
		return d.cards()
	}

	var deck [card.Count]card.Card
	for i := range deck {
		deck[i] = card.Card(i)
	}

	r := msRand{state: uint32(n)}
	var out [card.Count]card.Card
	for i := range out {
		left := card.Count - i
		j := r.next() % left
		out[i] = deck[j]
		deck[j] = deck[left-1]
	}
	return out
}

// Layout returns deal n arranged into its eight tableau columns, each listed
// from the top of the pile (dealt first) to the exposed card.
func Layout(n int) [Columns][]card.Card {
	var cols [Columns][]card.Card
	for i, c := range Cards(n) {
		cols[i%Columns] = append(cols[i%Columns], c)
	}
	return cols
}

// WriteRows writes deal n to w as it appears on the table: one line per row,
// cards separated by single spaces, the same form the published deal lists use.
func WriteRows(w io.Writer, n int) error {
	cards := Cards(n)
	var b strings.Builder
	for i, c := range cards {
		b.WriteString(c.String())
		if i%Columns == Columns-1 || i == len(cards)-1 {
			b.WriteByte('\n')
		} else {
			b.WriteByte(' ')
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
