package ui

import (
	"image"
	"image/color"

	"github.com/thomaslaurenson/cellmate/internal/card"
	"github.com/thomaslaurenson/cellmate/internal/deal"
	"github.com/thomaslaurenson/cellmate/internal/game"
	"github.com/thomaslaurenson/cellmate/internal/gfx"
)

// Colours of the Windows FreeCell table and chrome.
var (
	tableGreen = color.RGBA{0x00, 0x80, 0x00, 0xff}
	cellShadow = color.RGBA{0x00, 0x00, 0x00, 0xff}
	cellLight  = color.RGBA{0x00, 0xff, 0x00, 0xff}
)

// art holds the sprites the board is drawn with.
type art struct {
	cards  [card.Count]*gfx.Sprite
	king   *gfx.Sprite
	glyphs *gfx.Sprite
}

// newArt builds the sprites the table draws with.
func newArt() (*art, error) {
	cards, err := newCardSprites()
	if err != nil {
		return nil, err
	}
	return &art{cards: cards, king: newKingSprite(), glyphs: newGlyphStrip()}, nil
}

// fillRect draws a solid rectangle.
func (a *art) fillRect(dst surface, r image.Rectangle, c color.Color) {
	dst.FillRect(r, c)
}

// bevel draws a one-pixel border, with tl on the top and left edges and br on
// the bottom and right: the Windows convention for raised and sunken frames.
func (a *art) bevel(dst surface, r image.Rectangle, tl, br color.Color) {
	a.fillRect(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1), tl)
	a.fillRect(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y), tl)
	a.fillRect(dst, image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y), br)
	a.fillRect(dst, image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y), br)
}

// invertRect shows a card as selected by replacing the rectangle with its
// negative, which is how Windows FreeCell marked one.
func (a *art) invertRect(dst surface, r image.Rectangle) {
	dst.InvertRect(r)
}

func (a *art) drawTable(dst surface, t *table) {
	g := t.g
	dst.Fill(tableGreen)
	a.drawMenuBar(dst, t)

	for i := range 2 * game.Slots {
		a.bevel(dst, cellRect(i), cellShadow, cellLight)
	}
	if t.winTick == 0 {
		k := kingRect()
		a.drawKing(dst, k.Min.Add(k.Size().Div(2)), 1, t.cursor.X < screenW/2)
	}

	for i := range game.Slots {
		if c := g.Cell(i); c != card.None {
			a.drawCard(dst, c, cellRect(i).Min)
		}
		top := g.HomeTop(i)
		if f := t.flight; f != nil && f.move.To == (game.Pos{Area: game.Home, Index: i}) {
			top = below(top)
		}
		if top != card.None {
			a.drawCard(dst, top, cellRect(game.Slots+i).Min)
		}
	}

	for i := range deal.Columns {
		col := g.ColumnCards(i)
		for j, c := range col {
			a.drawCard(dst, c, cardPoint(i, j, len(col)))
		}
	}

	if p := t.selected; p != nil {
		n := len(g.ColumnCards(p.Index))
		pt := slotPoint(*p, n)
		a.invertRect(dst, image.Rect(pt.X, pt.Y, pt.X+cardW, pt.Y+cardH))
	}

	if r := t.reveal; r != nil {
		col := g.ColumnCards(r.column)
		a.drawCard(dst, col[r.index], cardPoint(r.column, r.index, len(col)))
	}

	if f := t.flight; f != nil {
		a.drawCard(dst, g.HomeTop(f.move.To.Index), f.at())
	}

	if t.winTick > 0 {
		centre, scale := winKing(t.winTick)
		a.drawKing(dst, centre, scale, false)
	}

	a.drawOpenMenu(dst, t)
	if t.dialog != nil {
		a.drawDialog(dst, t.dialog, t.edition, t.cursor)
	}
}

// below returns the card a home cell showed before c arrived on it.
func below(c card.Card) card.Card {
	if c == card.None || c.Rank() == card.Ace {
		return card.None
	}
	return card.New(c.Rank()-1, c.Suit())
}

// drawCard draws card c's face with its top left corner at pt.
func (a *art) drawCard(dst surface, c card.Card, pt image.Point) {
	face := a.cards[c]
	dst.Blit(face, face.Bounds(), pt)
}
