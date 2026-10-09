package card

import "testing"

func TestCard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		card Card
		str  string
		rank Rank
		suit Suit
		red  bool
	}{
		{name: "first card", card: 0, str: "AC", rank: Ace, suit: Clubs},
		{name: "ace of hearts", card: 2, str: "AH", rank: Ace, suit: Hearts, red: true},
		{name: "ten of diamonds", card: New(10, Diamonds), str: "TD", rank: 10, suit: Diamonds, red: true},
		{name: "last card", card: 51, str: "KS", rank: King, suit: Spades},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.card.String(); got != tc.str {
				t.Errorf("String() = %q, want %q", got, tc.str)
			}
			if got := tc.card.Rank(); got != tc.rank {
				t.Errorf("Rank() = %d, want %d", got, tc.rank)
			}
			if got := tc.card.Suit(); got != tc.suit {
				t.Errorf("Suit() = %d, want %d", got, tc.suit)
			}
			if got := tc.card.Red(); got != tc.red {
				t.Errorf("Red() = %v, want %v", got, tc.red)
			}
			parsed, err := Parse(tc.str)
			if err != nil || parsed != tc.card {
				t.Errorf("Parse(%q) = %v, %v; want %v", tc.str, parsed, err, tc.card)
			}
		})
	}
}

func TestParseRejects(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", "A", "ACE", "1C", "AX", "ac"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) returned no error", s)
		}
	}
}

func TestStringInvalid(t *testing.T) {
	t.Parallel()
	if got := Card(52).String(); got != "Card(52)" {
		t.Errorf("Card(52).String() = %q", got)
	}
}
