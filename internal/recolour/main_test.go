package main

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		colour color.RGBA
		want   string
		ok     bool
	}{
		{name: "the palette's own red", colour: rgb(0xff5555), want: "red", ok: true},
		{name: "the palette's own blue", colour: rgb(0x5555aa), want: "blue", ok: true},
		{name: "a red a channel off", colour: rgb(0xff5655), want: "red", ok: true},
		{name: "a blue a channel off", colour: rgb(0x5456aa), want: "blue", ok: true},
		{name: "a gold ten steps off", colour: rgb(0xfff555), want: "gold", ok: true},
		{name: "an almost-black", colour: rgb(0x000100), want: "black", ok: true},
		{name: "an almost-white", colour: rgb(0xfffdff), want: "white", ok: true},
		// A deck that has already been repainted must still be read, so
		// that running the command twice writes nothing the second time.
		{name: "the repainted red", colour: rgb(0xd0021b), want: "red", ok: true},
		{name: "the repainted blue", colour: rgb(0x1c3f94), want: "blue", ok: true},
		{name: "the repainted gold", colour: rgb(0xf2c200), want: "gold", ok: true},
		// A colour from some other deck is not guessed at.
		{name: "a green", colour: rgb(0x00ff00), ok: false},
		{name: "a mid grey", colour: rgb(0x808080), ok: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := snap(tc.colour)
			if ok != tc.ok {
				t.Fatalf("snap(%s) matched %v, want %v", hex(tc.colour), ok, tc.ok)
			}
			if ok && got.name != tc.want {
				t.Errorf("snap(%s) is %s, want %s", hex(tc.colour), got.name, tc.want)
			}
		})
	}
}

// TestPaletteEntriesAreTellingApart is the assumption snapDistance rests on:
// no two colours the command knows are close enough for a near miss at one
// to be mistaken for the other.
func TestPaletteEntriesAreTellingApart(t *testing.T) {
	t.Parallel()
	var known []color.RGBA
	for _, k := range palette {
		known = append(known, k.from, k.to)
	}
	for i, a := range known {
		for j, b := range known {
			if i >= j || a == b {
				continue
			}
			if d := channelDistance(a, b); d <= 2*snapDistance {
				t.Errorf("%s and %s are %d apart, too close for a snap distance of %d",
					hex(a), hex(b), d, snapDistance)
			}
		}
	}
}

func TestRepaint(t *testing.T) {
	t.Parallel()
	svg := []byte(`<path style="fill:#ff5555;stroke:#5456aa" /><path style="fill:#ffff55" />`)
	var found tally
	out, unknown := repaint(svg, &found)
	if len(unknown) > 0 {
		t.Fatalf("unknown colours %v", unknown)
	}
	want := `<path style="fill:#d0021b;stroke:#1c3f94" /><path style="fill:#f2c200" />`
	if string(out) != want {
		t.Errorf("repainted to\n%s\nwant\n%s", out, want)
	}
	if found.snapped != 1 {
		t.Errorf("counted %d near misses, want 1 for the stroke's blue", found.snapped)
	}
}

// TestRepaintRefusesAnotherDeck keeps the command from quietly rewriting
// drawings it does not know, which is the one way it could damage a deck.
func TestRepaintRefusesAnotherDeck(t *testing.T) {
	t.Parallel()
	var found tally
	_, unknown := repaint([]byte(`<path style="fill:#123456" />`), &found)
	if len(unknown) != 1 || unknown[0] != "#123456" {
		t.Errorf("reported %v, want the one colour it does not know", unknown)
	}
}

func TestRunRepaintsAndIsRepeatable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "AH.svg")
	const before = `<svg><path style="fill:#ff5555;stroke:#000100" /></svg>`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := run(dir, false, &out); err != nil {
		t.Fatalf("first run: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const want = `<svg><path style="fill:#d0021b;stroke:#000000" /></svg>`
	if string(got) != want {
		t.Errorf("repainted to %s, want %s", got, want)
	}

	// A second run finds the deck already painted and leaves it alone.
	out.Reset()
	if err := run(dir, false, &out); err != nil {
		t.Fatalf("second run: %v", err)
	}
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != want {
		t.Errorf("second run changed the drawing to %s", again)
	}
	if !strings.Contains(out.String(), "0 repainted") {
		t.Errorf("second run reported %q, want it to repaint nothing", out.String())
	}
}

// TestRunDryWritesNothing checks the flag that lets a run be inspected
// before it touches the deck.
func TestRunDryWritesNothing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "AH.svg")
	const before = `<svg><path style="fill:#ff5555" /></svg>`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := run(dir, true, &out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != before {
		t.Errorf("dry run wrote %s", got)
	}
	if !strings.Contains(out.String(), "dry run") {
		t.Errorf("dry run reported %q, want it to say so", out.String())
	}
}

// TestRunNeedsDrawings reports an empty or wrong directory rather than
// succeeding at doing nothing.
func TestRunNeedsDrawings(t *testing.T) {
	t.Parallel()
	var out strings.Builder
	if err := run(t.TempDir(), false, &out); err == nil {
		t.Error("run on an empty directory succeeded")
	}
}
