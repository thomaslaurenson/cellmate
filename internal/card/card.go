// Package card defines the 52 playing cards in the order the Microsoft deal
// algorithm numbers them.
package card

import (
	"fmt"
	"strings"
)

// Suit is a card suit, in the order the Microsoft deck interleaves them.
type Suit int

// The suits, in deck order.
const (
	Clubs Suit = iota
	Diamonds
	Hearts
	Spades
)

// Rank is a card rank from Ace (1) to King (13).
type Rank int

// Named ranks. Ranks 2 to 10 are their own value.
const (
	Ace   Rank = 1
	Jack  Rank = 11
	Queen Rank = 12
	King  Rank = 13
)

// Count is the number of cards in a deck.
const Count = 52

const (
	rankLetters = "A23456789TJQK"
	suitLetters = "CDHS"
)

// None marks an empty slot, such as a free cell with no card in it.
const None Card = -1

// Card is one of the 52 cards, numbered 0 to 51 as the Microsoft deck orders
// them: rank-major, so 0 is the Ace of Clubs, 1 the Ace of Diamonds and 51
// the King of Spades.
type Card int

// New returns the card of rank r and suit s.
func New(r Rank, s Suit) Card {
	return Card((int(r)-1)*4 + int(s))
}

// Rank returns the card's rank.
func (c Card) Rank() Rank { return Rank(int(c)/4 + 1) }

// Suit returns the card's suit.
func (c Card) Suit() Suit { return Suit(int(c) % 4) }

// Red reports whether the card is a Diamond or a Heart.
func (c Card) Red() bool {
	s := c.Suit()
	return s == Diamonds || s == Hearts
}

// Valid reports whether c names one of the 52 cards.
func (c Card) Valid() bool { return c >= 0 && c < Count }

// String returns the two-letter form used by FreeCell tools, such as "TD"
// for the Ten of Diamonds.
func (c Card) String() string {
	if !c.Valid() {
		return fmt.Sprintf("Card(%d)", int(c))
	}
	return string([]byte{rankLetters[c.Rank()-1], suitLetters[c.Suit()]})
}

// Parse reads the two-letter form written by String.
func Parse(s string) (Card, error) {
	if len(s) != 2 {
		return 0, fmt.Errorf("parse card %q: want two letters", s)
	}
	r := strings.IndexByte(rankLetters, s[0])
	u := strings.IndexByte(suitLetters, s[1])
	if r < 0 || u < 0 {
		return 0, fmt.Errorf("parse card %q: unknown rank or suit", s)
	}
	return New(Rank(r+1), Suit(u)), nil
}
