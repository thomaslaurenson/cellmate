// Command recolour repaints the card drawings onto a proper card palette.
//
// The deck was cut from a drawing made in a sixteen-colour EGA palette, so
// its red is a washed-out salmon and its court cards are robed in
// periwinkle. The cards are rasterised as large as the window allows, and
// at that size the palette reads as a computer's idea of a deck rather than
// a deck. This maps each colour the deck is drawn in onto the colour a
// printed card would use.
//
// The pass is a command rather than a hand edit because the deck is 52
// files, and because the drawings carry colours a channel or two off the
// palette, #5456aa beside #5555aa and seven shades of almost-black, that a
// search and replace would leave behind as flecks of the old colour. Every
// colour is snapped onto the palette entry it is a hair away from before it
// is repainted, and one that is near no entry at all stops the run rather
// than being quietly rewritten.
//
//	go run ./internal/recolour -dir internal/ui/cards
package main

import (
	"flag"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ink is one colour of the deck: the palette colour the drawings are made
// of, and the colour they are repainted in.
type ink struct {
	name     string
	from, to color.RGBA
}

// palette is the whole of the deck's colouring. Black and white are listed
// although neither changes, because a colour matching no entry stops the
// run and the drawings are mostly black and white.
var palette = []ink{
	{"black", rgb(0x000000), rgb(0x000000)},
	{"white", rgb(0xffffff), rgb(0xffffff)},
	{"red", rgb(0xff5555), rgb(0xd0021b)},
	{"blue", rgb(0x5555aa), rgb(0x1c3f94)},
	{"gold", rgb(0xffff55), rgb(0xf2c200)},
}

// snapDistance is how far off a palette entry a colour may be and still be
// taken for it, measured as the largest difference on any one channel. The
// drawings stray a few steps at most, and no two palette entries are closer
// than 170 apart, so no distance this small can pick the wrong entry.
const snapDistance = 16

// colourRE matches a colour in a drawing. Every "#" in the deck introduces
// one, written as six hex digits, and the word boundary keeps a longer run
// of hex from being read as a colour and a stray digit.
var colourRE = regexp.MustCompile(`(?i)#[0-9a-f]{6}\b`)

func main() {
	dir := flag.String("dir", filepath.Join("internal", "ui", "cards"), "directory of card drawings")
	dry := flag.Bool("n", false, "report what would change without writing anything")
	flag.Parse()
	if err := run(*dir, *dry, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "[!] %v\n", err)
		os.Exit(1)
	}
}

// run repaints every drawing in dir, reporting what it found and what it
// wrote to w.
func run(dir string, dry bool, w io.Writer) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.svg"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no drawings in %s", dir)
	}
	sort.Strings(files)

	var found tally
	changed := 0
	for _, path := range files {
		svg, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out, unknown := repaint(svg, &found)
		if len(unknown) > 0 {
			return fmt.Errorf("%s: %s is not a colour this deck is drawn in",
				filepath.Base(path), strings.Join(unknown, ", "))
		}
		if string(out) == string(svg) {
			continue
		}
		changed++
		if dry {
			continue
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return err
		}
	}

	note := ""
	if dry {
		note = " (dry run)"
	}
	fmt.Fprintf(w, "%d drawings, %d repainted%s\n", len(files), changed, note)
	found.report(w)
	return nil
}

// tally counts what a run found, so it can report what it did rather than
// only that it did something.
type tally struct {
	// uses counts the colours repainted, by palette entry name.
	uses map[string]int
	// snapped counts those that were a near miss rather than the palette
	// colour itself: the flecks a search and replace would have left behind.
	snapped int
}

func (t *tally) count(name string, exact bool) {
	if t.uses == nil {
		t.uses = map[string]int{}
	}
	t.uses[name]++
	if !exact {
		t.snapped++
	}
}

// report prints each colour of the deck before and after.
func (t *tally) report(w io.Writer) {
	fmt.Fprintf(w, "\n%-6s %-9s %-9s %s\n", "ink", "before", "after", "uses")
	for _, k := range palette {
		fmt.Fprintf(w, "%-6s %-9s %-9s %d\n", k.name, hex(k.from), hex(k.to), t.uses[k.name])
	}
	if t.snapped > 0 {
		fmt.Fprintf(w, "\n%d of those were near misses snapped onto the palette first\n", t.snapped)
	}
}

// repaint rewrites every colour in svg, counting what it found by palette
// entry. It returns the colours that matched no entry, which the caller
// treats as a deck this command should not touch.
func repaint(svg []byte, found *tally) ([]byte, []string) {
	var unknown []string
	out := colourRE.ReplaceAllFunc(svg, func(m []byte) []byte {
		c, ok := parseHex(string(m))
		if !ok {
			unknown = append(unknown, string(m))
			return m
		}
		k, ok := snap(c)
		if !ok {
			unknown = append(unknown, string(m))
			return m
		}
		found.count(k.name, c == k.from || c == k.to)
		return []byte(hex(k.to))
	})
	return out, unknown
}

// snap returns the palette entry c is drawn as, taking a colour a channel
// or two off an entry for that entry.
//
// A colour already repainted matches the entry it was painted in, so a
// second run over the same deck finds every colour and writes nothing.
// That is what lets the pass be run again after the deck is re-cut from its
// source without having to know whether it has been run before.
func snap(c color.RGBA) (ink, bool) {
	for _, k := range palette {
		if channelDistance(c, k.from) <= snapDistance || channelDistance(c, k.to) <= snapDistance {
			return k, true
		}
	}
	return ink{}, false
}

// channelDistance is the largest difference between a and b on any one
// channel, which is the measure that catches a colour nudged on a single
// channel however small the nudge is elsewhere.
func channelDistance(a, b color.RGBA) int {
	return max(diff(a.R, b.R), diff(a.G, b.G), diff(a.B, b.B))
}

func diff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// parseHex reads a "#rrggbb" colour.
func parseHex(s string) (color.RGBA, bool) {
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	return rgb(uint32(v)), true
}

// hex writes a colour the way the drawings do, which is how a repainted
// file stays a one-line diff against the one it replaced.
func hex(c color.RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// rgb builds an opaque colour from a 0xrrggbb literal.
func rgb(v uint32) color.RGBA {
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}
