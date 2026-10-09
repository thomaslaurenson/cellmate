package ui

import (
	"bytes"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/card"
)

// TestEveryCardHasItsOwnFace checks the deck by its ink. The drawings use a
// single red, so a pip card of a red suit has it and one of a black suit does
// not, and a deck with its files misnamed, or of another pattern, fails here
// rather than dealing the wrong cards. The court cards are pictures drawn in
// every colour whatever their suit, so only the pip cards are checked.
func TestEveryCardHasItsOwnFace(t *testing.T) {
	t.Parallel()
	const redInk = "#d0021b"
	seen := make(map[string]card.Card, card.Count)
	for c := card.Card(0); c < card.Count; c++ {
		name := cardFile(c)
		if other, ok := seen[name]; ok {
			t.Errorf("%v and %v share the face %s", c, other, name)
		}
		seen[name] = c

		svg, err := cardArt.ReadFile(name)
		if err != nil {
			t.Errorf("%v: %v", c, err)
			continue
		}
		if !bytes.Contains(svg, []byte("<svg")) {
			t.Errorf("%s is not an SVG drawing", name)
		}
		if c.Rank() >= card.Jack {
			continue
		}
		if got := bytes.Contains(svg, []byte(redInk)); got != c.Red() {
			t.Errorf("%v has red ink %v, want %v", c, got, c.Red())
		}
	}
}
