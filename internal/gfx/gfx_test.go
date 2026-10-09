package gfx

import (
	"image"
	"testing"
)

func TestInputZeroValueReportsNothingHappening(t *testing.T) {
	t.Parallel()
	var in Input

	if in.KeyDown(KeyF2) || in.KeyPressed(KeyF2) || in.KeyTicks(KeyF2) != 0 {
		t.Error("the zero Input reports a key down")
	}
	if in.ButtonDown(ButtonLeft) || in.ButtonPressed(ButtonLeft) || in.ButtonReleased(ButtonLeft) {
		t.Error("the zero Input reports a button down")
	}
	if in.Cursor != (image.Point{}) {
		t.Errorf("cursor = %v, want the origin", in.Cursor)
	}
}

func TestInputKeyState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		ticks       int
		wantDown    bool
		wantPressed bool
	}{
		{name: "not held", ticks: 0},
		{name: "went down this tick", ticks: 1, wantDown: true, wantPressed: true},
		{name: "still held", ticks: 2, wantDown: true},
		{name: "held a long time", ticks: 400, wantDown: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := Input{Keys: map[Key]int{KeyBackspace: tc.ticks}}

			if got := in.KeyDown(KeyBackspace); got != tc.wantDown {
				t.Errorf("KeyDown = %v, want %v", got, tc.wantDown)
			}
			if got := in.KeyPressed(KeyBackspace); got != tc.wantPressed {
				t.Errorf("KeyPressed = %v, want %v", got, tc.wantPressed)
			}
			if got := in.KeyTicks(KeyBackspace); got != tc.ticks {
				t.Errorf("KeyTicks = %d, want %d", got, tc.ticks)
			}
		})
	}
}

func TestInputButtonState(t *testing.T) {
	t.Parallel()
	// A click that began and ended inside one tick reports both edges while
	// the button is no longer down, which is the case the counting exists
	// for and the one a sampled implementation loses.
	in := Input{
		Held:     map[Button]bool{ButtonLeft: false},
		Pressed:  map[Button]bool{ButtonLeft: true},
		Released: map[Button]bool{ButtonLeft: true},
	}
	if in.ButtonDown(ButtonLeft) {
		t.Error("ButtonDown is true after the release")
	}
	if !in.ButtonPressed(ButtonLeft) || !in.ButtonReleased(ButtonLeft) {
		t.Error("a press and release inside one tick was not reported")
	}
}

// TestKeyNoneIsTheZeroValue guards the convention the table relies on: a
// field holding "the key pressed this tick" is empty until one is.
func TestKeyNoneIsTheZeroValue(t *testing.T) {
	t.Parallel()
	var k Key
	if k != KeyNone {
		t.Errorf("the zero Key is %d, want KeyNone", k)
	}
	if KeyNone == KeyF1 {
		t.Error("KeyNone collides with a real key")
	}
}
