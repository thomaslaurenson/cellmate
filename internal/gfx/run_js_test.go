//go:build js && wasm

package gfx

import (
	"image"
	"testing"
)

// newTestDriver builds a driver with the maps Run would have made for it,
// but no canvas. Nothing below touches the page, so none is needed.
func newTestDriver() *driver {
	d := &driver{
		cfg:        Config{Width: 640, Height: 480},
		keyHeld:    make(map[Key]bool),
		keyTicks:   make(map[Key]int),
		keyPresses: make(map[Key]int),
		held:       make(map[Button]bool),
		presses:    make(map[Button]int),
		releases:   make(map[Button]int),
	}
	d.tickInput = Input{
		Held:     make(map[Button]bool),
		Pressed:  make(map[Button]bool),
		Released: make(map[Button]bool),
		Keys:     d.keyTicks,
	}
	for _, b := range []Button{ButtonLeft, ButtonRight} {
		d.held[b], d.presses[b], d.releases[b] = false, 0, 0
	}
	return d
}

func TestDriverReportsAFingerTapAsALeftClick(t *testing.T) {
	t.Parallel()
	d := newTestDriver()
	d.finger.begin(image.Pt(120, 240))
	d.tick()
	d.finger.end()

	press := d.tick()
	if !press.ButtonPressed(ButtonLeft) {
		t.Fatal("a tap did not press the left button")
	}
	if press.Cursor != image.Pt(120, 240) {
		t.Errorf("cursor = %v, want the tap point", press.Cursor)
	}
	release := d.tick()
	if !release.ButtonReleased(ButtonLeft) {
		t.Fatal("the tap never released the left button")
	}
	if release.ButtonPressed(ButtonLeft) {
		t.Error("the release tick pressed the button a second time")
	}

	idle := d.tick()
	if idle.ButtonPressed(ButtonLeft) || idle.ButtonReleased(ButtonLeft) ||
		idle.ButtonDown(ButtonLeft) {
		t.Errorf("the tick after a tap still reported the left button: %+v", idle)
	}
}

func TestDriverReportsAFingerHoldAsTheRightButton(t *testing.T) {
	t.Parallel()
	d := newTestDriver()
	d.finger.begin(image.Pt(50, 60))

	for range holdTicks - 1 {
		if d.tick().ButtonDown(ButtonRight) {
			t.Fatal("the hold reported the right button before the delay was up")
		}
	}
	held := d.tick()
	if !held.ButtonDown(ButtonRight) {
		t.Fatal("the hold did not report the right button")
	}
	if held.ButtonDown(ButtonLeft) {
		t.Error("the hold also reported the left button")
	}

	d.finger.end()
	after := d.tick()
	if after.ButtonDown(ButtonRight) {
		t.Error("the right button stayed down after the finger lifted")
	}
	if after.ButtonPressed(ButtonLeft) {
		t.Error("lifting after a hold also clicked")
	}
}

func TestDriverCombinesAMouseAndAFinger(t *testing.T) {
	t.Parallel()
	d := newTestDriver()
	// A mouse button held while a finger taps: neither may cancel the other,
	// because a laptop with a touch screen has both.
	d.held[ButtonRight] = true
	d.finger.begin(image.Pt(10, 20))
	d.tick()
	d.finger.end()

	got := d.tick()
	if !got.ButtonDown(ButtonRight) {
		t.Error("the finger's tap cleared the mouse's right button")
	}
	if !got.ButtonPressed(ButtonLeft) {
		t.Error("the mouse's held button swallowed the finger's tap")
	}
}

func TestDriverClearsAMouseButtonOnceReleased(t *testing.T) {
	t.Parallel()
	d := newTestDriver()
	d.held[ButtonLeft] = true
	d.presses[ButtonLeft] = 1
	if got := d.tick(); !got.ButtonDown(ButtonLeft) || !got.ButtonPressed(ButtonLeft) {
		t.Fatalf("the press was not reported: %+v", got)
	}
	// The press is an edge, so it lasts one tick while the button stays down
	if got := d.tick(); got.ButtonPressed(ButtonLeft) {
		t.Error("the press was reported on a second tick")
	} else if !got.ButtonDown(ButtonLeft) {
		t.Error("the button stopped being held while it was still down")
	}

	d.held[ButtonLeft] = false
	d.releases[ButtonLeft] = 1
	if got := d.tick(); !got.ButtonReleased(ButtonLeft) || got.ButtonDown(ButtonLeft) {
		t.Errorf("the release was not reported: %+v", got)
	}
}

func TestDriverKeyHeldAcrossTicks(t *testing.T) {
	t.Parallel()
	d := newTestDriver()
	d.keyHeld[KeyBackspace] = true
	d.keyPresses[KeyBackspace] = 1

	for want := 1; want <= 3; want++ {
		if got := d.tick().KeyTicks(KeyBackspace); got != want {
			t.Fatalf("tick %d reported the key held for %d ticks", want, got)
		}
	}
	d.keyHeld[KeyBackspace] = false
	if got := d.tick().KeyTicks(KeyBackspace); got != 0 {
		t.Errorf("the released key still reported %d ticks", got)
	}
}

func TestDriverReportsAKeystrokeInsideOneTick(t *testing.T) {
	t.Parallel()
	d := newTestDriver()
	// Pressed and released between two ticks, which a sampled reading loses
	d.keyHeld[KeyF2] = false
	d.keyPresses[KeyF2] = 1

	if !d.tick().KeyPressed(KeyF2) {
		t.Fatal("a keystroke inside one tick was not reported")
	}
	if d.tick().KeyDown(KeyF2) {
		t.Error("the key stayed down after the tick it was reported on")
	}
}
