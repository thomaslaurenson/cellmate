# cellmate

![Release Build](https://img.shields.io/github/actions/workflow/status/thomaslaurenson/cellmate/tag.yml?style=flat&label=release&logo=github) ![Main Build](https://img.shields.io/github/actions/workflow/status/thomaslaurenson/cellmate/main.yml?style=flat&label=main&logo=github)

![Release Version](https://img.shields.io/github/v/release/thomaslaurenson/cellmate?style=flat&logo=github) ![Release downloads](https://img.shields.io/github/downloads/thomaslaurenson/cellmate/total?style=flat&label=downloads&logo=github)

![Go Version](https://img.shields.io/github/go-mod/go-version/thomaslaurenson/cellmate?style=flat&logo=go) ![Code Coverage](https://img.shields.io/badge/Coverage-90.7%25-blue?style=flat&logo=go)

FreeCell in the style of the Windows 95 and XP versions, provided as a single multi-platform binary or [playable in the browser](https://cellmate.thomaslaurenson.com/).

## What

- Reproduces the numbered Microsoft deals, so deal 617 here is deal 617 in Windows
- Two editions: `95` offers 32,000 deals, `xp` extends the range to 1,000,000 deals
- The two deals that cannot be won, which Windows hid at -1 and -2 as a joke on its help file's promise that every deal can be; as there, they count for nothing in the statistics
- A desktop window built with [Ebitengine](https://ebitengine.org), and the same game compiled to WebAssembly, where it draws on the page's own canvas and carries no game engine at all
- Cards drawn from vector art at the size they are shown, so they stay sharp in any window and on high-resolution screens
- Plays with a mouse, the keyboard alone, or a finger (tap to pick a card, press and hold to look under one)
- A `deal` subcommand prints any deal as text, for solvers and scripts

## Installation

Download a pre-built binary from the [releases page](https://github.com/thomaslaurenson/cellmate/releases). For easier install, use the bash installer script:

```sh
curl -fsSL https://github.com/thomaslaurenson/cellmate/releases/latest/download/install.sh | bash
```

Or the PowerShell installer script if on Windows:

```powershell
irm https://github.com/thomaslaurenson/cellmate/releases/latest/download/install.ps1 | iex
```

Install from source:

```sh
go install github.com/thomaslaurenson/cellmate@latest
```

On Linux the window needs the X11 and OpenGL runtime libraries, which every desktop install already has.

## Usage

```sh
cellmate                      # open a random deal
cellmate --game 617           # open deal 617
cellmate --game -1            # open one of the two deals that cannot be won
cellmate --edition 95         # Windows 95 rules and deal range
cellmate deal 617             # print deal 617, one row per line
cellmate deal -- -1           # print an impossible deal; its minus needs the --
cellmate version              # print the version
cellmate completion bash      # print a shell completion script
```

In the [browser](https://cellmate.thomaslaurenson.com/), the page URL takes the same options: `?game=617&edition=95`.

| Input | Action |
|---|---|
| Click a card, then a destination | Move it; runs move together when the free cells allow |
| Double-click a column | Send its exposed card to a free cell |
| Hold the right button on a card | Show a buried card in full |
| F1 | How to play |
| F2 | New game |
| F3 | Select a game by number |
| F4 | Statistics |
| F5 | Options |
| F10 | Undo (XP edition only) |
| Esc | Cancel the selection, or close a menu or message |

The keyboard alone plays too, as it did in Windows: the digits `1` to `8` name the columns, `0` the free cells and `9` the home cells. Type where a card comes from, then where it goes: `10` sends column 1's card to a free cell, `0` then `3` brings a free cell card to column 3, and pressing `0` again moves on to the next free cell. When a run moves to an empty column, `C` moves the column and `S` a single card.

Cards that can no longer be needed go home on their own. Leaving a game after moving a card counts as a loss, so cellmate asks first.

Windows showed the game number in the title bar, and so does cellmate. A tiled window or a phone's home screen has no title bar, so the Options box can also show it on the table beside the cards left.

Options and statistics are kept between runs: in `cellmate/cellmate.json` under the user configuration directory on the desktop (`~/.config` on Linux, `~/Library/Application Support` on macOS, `%AppData%` on Windows), and in the browser's local storage on the web. The edition chosen in Options is used unless `--edition` or `?edition=` says otherwise.

## Building

```sh
make build        # desktop binary in dist/
make build_wasm   # browser build in dist/web/
make serve_wasm   # build and serve the browser version on localhost:8080
make ci           # run every check CI runs
```

Building needs only a Go toolchain; `make ci` also runs the browser build's tests under Node. `make serve_wasm` serves the browser build with a Go program under `internal/`.

## Credits

- Paul Alfille created FreeCell on the PLATO system in 1978
- Jim Horne wrote the Microsoft version and published its deal algorithm
- The card faces are Dmitry Fomin's [English pattern playing cards deck](https://commons.wikimedia.org/wiki/File:English_pattern_playing_cards_deck.svg) from Wikimedia Commons
- The text is set in Go Mono, one of the [Go fonts](https://go.dev/blog/go-fonts) by Bigelow & Holmes
- Everything else, including the king, is cellmate's own
