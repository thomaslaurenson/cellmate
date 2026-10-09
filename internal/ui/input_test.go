package ui

import (
	"image"
	"slices"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/gfx"
)

func TestReadInputCarriesTheCursorAndButtons(t *testing.T) {
	t.Parallel()
	raw := gfx.Input{
		Cursor:   image.Pt(120, 240),
		Held:     map[gfx.Button]bool{gfx.ButtonRight: true},
		Pressed:  map[gfx.Button]bool{gfx.ButtonLeft: true},
		Released: map[gfx.Button]bool{gfx.ButtonLeft: true},
	}
	in := readInput(raw)

	if in.cursor != image.Pt(120, 240) {
		t.Errorf("cursor = %v, want (120,240)", in.cursor)
	}
	if !in.leftPressed || !in.leftReleased {
		t.Error("a press and release in one tick did not reach the table")
	}
	if !in.rightDown {
		t.Error("the held right button did not reach the table")
	}
}

func TestReadInputMapsEveryFunctionKey(t *testing.T) {
	t.Parallel()
	// Each menu shortcut must survive the trip from the platform to the
	// table. A key missing from the map reads as no key at all, which is
	// silent: the menu item simply stops responding.
	for _, k := range functionKeys {
		t.Run(keyName(k), func(t *testing.T) {
			t.Parallel()
			in := readInput(gfx.Input{Keys: map[gfx.Key]int{k: 1}})
			if in.key != k {
				t.Errorf("key = %v, want %v", in.key, k)
			}
		})
	}
}

func TestReadInputReportsNoKeyWhenNonePressed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  gfx.Input
	}{
		{name: "nothing at all", raw: gfx.Input{}},
		{name: "a function key merely held", raw: gfx.Input{Keys: map[gfx.Key]int{gfx.KeyF2: 9}}},
		{name: "a key the table does not read", raw: gfx.Input{Keys: map[gfx.Key]int{gfx.KeyShift: 1}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := readInput(tc.raw).key; got != gfx.KeyNone {
				t.Errorf("key = %v, want KeyNone", got)
			}
		})
	}
}

func TestReadInputTreatsBothEnterKeysAlike(t *testing.T) {
	t.Parallel()
	for _, k := range []gfx.Key{gfx.KeyEnter, gfx.KeyNumpadEnter} {
		t.Run(keyName(k), func(t *testing.T) {
			t.Parallel()
			if !readInput(gfx.Input{Keys: map[gfx.Key]int{k: 1}}).enter {
				t.Error("enter was not reported")
			}
		})
	}
}

func TestReadInputReportsModifiersWhileHeld(t *testing.T) {
	t.Parallel()
	// A modifier is read as held rather than pressed, so it still counts on
	// the tick a function key lands.
	raw := gfx.Input{Keys: map[gfx.Key]int{
		gfx.KeyControl: 40,
		gfx.KeyShift:   40,
		gfx.KeyF10:     1,
	}}
	in := readInput(raw)

	if !in.ctrl || !in.shift {
		t.Errorf("ctrl = %v, shift = %v, want both held", in.ctrl, in.shift)
	}
	if in.key != gfx.KeyF10 {
		t.Errorf("key = %v, want F10 alongside the modifiers", in.key)
	}
}

func TestReadInputCopiesTypedCharacters(t *testing.T) {
	t.Parallel()
	chars := []rune("617")
	in := readInput(gfx.Input{Chars: chars})

	if !slices.Equal(in.chars, chars) {
		t.Fatalf("chars = %q, want %q", string(in.chars), string(chars))
	}
	// The platform reuses its buffer between ticks, so the table must hold a
	// copy rather than a view that the next tick overwrites.
	chars[0] = 'x'
	if in.chars[0] == 'x' {
		t.Error("the table shares the platform's character buffer")
	}
}

func TestRepeatingRepeatsTheWayATextBoxDoes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		ticks int
		want  bool
	}{
		{name: "not held", ticks: 0},
		{name: "first tick fires at once", ticks: 1, want: true},
		{name: "silent during the delay", ticks: 2},
		{name: "still silent at the end of the delay", ticks: 30},
		{name: "repeats after the delay", ticks: 33, want: true},
		{name: "between repeats", ticks: 34},
		{name: "next repeat", ticks: 36, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw := gfx.Input{Keys: map[gfx.Key]int{gfx.KeyBackspace: tc.ticks}}
			if got := repeating(raw, gfx.KeyBackspace); got != tc.want {
				t.Errorf("repeating at %d ticks = %v, want %v", tc.ticks, got, tc.want)
			}
		})
	}
}

func TestReadInputBackspaceUsesTheRepeatRule(t *testing.T) {
	t.Parallel()
	held := readInput(gfx.Input{Keys: map[gfx.Key]int{gfx.KeyBackspace: 2}})
	if held.backspace {
		t.Error("backspace fired during the repeat delay")
	}
	first := readInput(gfx.Input{Keys: map[gfx.Key]int{gfx.KeyBackspace: 1}})
	if !first.backspace {
		t.Error("backspace did not fire on the tick it went down")
	}
}

// keyName gives a subtest a readable name for a key.
func keyName(k gfx.Key) string {
	names := map[gfx.Key]string{
		gfx.KeyF1: "F1", gfx.KeyF2: "F2", gfx.KeyF3: "F3", gfx.KeyF4: "F4",
		gfx.KeyF5: "F5", gfx.KeyF10: "F10", gfx.KeyEnter: "Enter",
		gfx.KeyNumpadEnter: "NumpadEnter", gfx.KeyEscape: "Escape",
		gfx.KeyBackspace: "Backspace", gfx.KeyControl: "Control",
		gfx.KeyShift: "Shift",
	}
	if n, ok := names[k]; ok {
		return n
	}
	return "unnamed"
}
