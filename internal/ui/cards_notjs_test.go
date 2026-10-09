//go:build !(js && wasm)

package ui

import (
	"image"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/gfx"
)

// TestCardFacesRasterise draws every face the way the desktop build does and
// checks the result looks like a card: mostly white, with red ink on the red
// suits and none on the black. Parsing a drawing does not prove the
// rasteriser can draw it; this does.
func TestCardFacesRasterise(t *testing.T) {
	t.Parallel()
	for c := card.Card(0); c < card.Count; c++ {
		svg, err := cardArt.ReadFile(cardFile(c))
		if err != nil {
			t.Fatalf("%v: %v", c, err)
		}
		face, err := gfx.RenderSVG(svg, image.Pt(cardW, cardH))
		if err != nil {
			t.Errorf("%v: %v", c, err)
			continue
		}
		var white, red int
		for y := range cardH {
			for x := range cardW {
				p := face.RGBAAt(x, y)
				switch {
				case p.A < 0x80:
				case p.R > 0xe0 && p.G > 0xe0 && p.B > 0xe0:
					white++
				case p.R > 0xc0 && p.G < 0x80 && p.B < 0x80:
					red++
				}
			}
		}
		// Every face shows some white card. The court cards are mostly
		// picture and drawn in red whatever their suit, so the rest of the
		// checks are for the pip cards alone.
		if white*5 < cardW*cardH {
			t.Errorf("%v is %d white pixels of %d, want a card face", c, white, cardW*cardH)
		}
		if c.Rank() >= card.Jack {
			continue
		}
		if white*2 < cardW*cardH {
			t.Errorf("%v is %d white pixels of %d, want a mostly white card", c, white, cardW*cardH)
		}
		if c.Red() && red == 0 {
			t.Errorf("%v has no red on it", c)
		}
		if !c.Red() && red != 0 {
			t.Errorf("%v has %d red pixels, want a black suit", c, red)
		}
	}
}

// TestNewCardSpritesMakesEveryCard is the start-up path itself: every
// drawing parses, and each card gets a sprite of its own.
func TestNewCardSpritesMakesEveryCard(t *testing.T) {
	t.Parallel()
	sprites, err := newCardSprites()
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[*gfx.Sprite]bool, len(sprites))
	for c, s := range sprites {
		if s == nil {
			t.Fatalf("%v has no sprite", card.Card(c))
		}
		if seen[s] {
			t.Errorf("%v shares a sprite with another card", card.Card(c))
		}
		seen[s] = true
		if got := s.Bounds(); got != image.Rect(0, 0, cardW, cardH) {
			t.Errorf("%v is %v, want a %dx%d card", card.Card(c), got, cardW, cardH)
		}
	}
}
