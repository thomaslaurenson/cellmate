# Card faces

The 52 card faces as separate SVG files, which both builds embed and draw from. Each face is rasterised at the size it is shown at, so the cards stay sharp however large the window is and on high-resolution screens: the desktop build rasterises them with oksvg, and the browser build hands them to the browser.

## Where they came from

The faces are cut from Dmitry Fomin's [English pattern playing cards deck][commons] on Wikimedia Commons, one SVG holding all 52 cards. Each file here is one card's drawing, unchanged apart from four things:

- It is moved so the card's top left corner is at the origin, in a 360 by 540 `viewBox`.
- Inkscape's editor metadata (its `inkscape:` and `sodipodi:` attributes and elements) is removed; none of it affects how the card looks.
- It is named the way the game writes a card: rank then suit, with ten as `T`, so `TD.svg` is the ten of diamonds and `AS.svg` the ace of spades.
- It is repainted onto a card palette, described below.

## The palette

The deck was drawn in a sixteen-colour EGA palette, which is why its red is the salmon `#ff5555` and its court cards are robed in the periwinkle `#5555aa`. The faces are rasterised as large as the window allows, and at that size the palette reads as a computer's idea of a deck. Each colour is repainted as a printed card's:

| ink | drawn as | printed as |
|---|---|---|
| black | `#000000` | `#000000` |
| white | `#ffffff` | `#ffffff` |
| red | `#ff5555` | `#d0021b` |
| blue | `#5555aa` | `#1c3f94` |
| gold | `#ffff55` | `#f2c200` |

The repainting is `make gen_card_colours`, not a hand edit, because the deck is 52 files and carried colours a channel or two off the palette, `#5456aa` beside `#5555aa` and seven shades of almost-black, that a search and replace would have left behind as flecks of the old colour. The command snaps every colour onto the palette entry it is a hair away from before repainting it, refuses a colour it does not recognise rather than guessing, and leaves a deck it has already painted alone, so it can be run again after a card is re-cut from source. `make check_embed` fails if any drawing has drifted off the palette.

## The shape

The drawings are 2 by 3 and the game's card slot is `cardW` by `cardH` in `../layout.go`, which is 64 by 96 and so the same shape. The faces are drawn to fill the slot, so a slot of any other shape would stretch every one of them: the original game's cards were 71 by 96 and the faces were stretched a little wider than drawn to fit. `TestCardIsTheShapeItIsDrawn` keeps the two in step.

## Licence

Wikimedia Commons lists the deck as released into the public domain by its author.

[commons]: https://commons.wikimedia.org/wiki/File:English_pattern_playing_cards_deck.svg
