package ui

import (
	"embed"
	"fmt"
	"image"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/gfx"
)

// cardArt holds the 52 card faces as SVG drawings: Dmitry Fomin's English
// pattern deck, one file per card, named the way the game writes a card so
// that TD.svg is the ten of diamonds. Both builds embed them and draw each
// face at the size it is shown, which is what keeps the cards sharp however
// large the window is.
//
//go:embed cards/*.svg
var cardArt embed.FS

// cardFile returns the drawing of card c.
func cardFile(c card.Card) string { return "cards/" + c.String() + ".svg" }

// newCardSprites makes a sprite of every card's face. The drawings are part
// of the binary and every one is read by the tests, so one that cannot be
// read is a build that should never have shipped, and stops the game rather
// than dealing blank cards.
func newCardSprites() ([card.Count]*gfx.Sprite, error) {
	var sprites [card.Count]*gfx.Sprite
	for c := card.Card(0); c < card.Count; c++ {
		svg, err := cardArt.ReadFile(cardFile(c))
		if err != nil {
			return sprites, fmt.Errorf("load card face %q: %w", cardFile(c), err)
		}
		s, err := gfx.NewVectorSprite(svg, image.Pt(cardW, cardH))
		if err != nil {
			return sprites, fmt.Errorf("load card face %q: %w", cardFile(c), err)
		}
		sprites[c] = s
	}
	return sprites, nil
}
