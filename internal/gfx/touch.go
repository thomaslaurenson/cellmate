package gfx

import (
	"image"
	"math"
)

// Timing and distance for a finger, in ticks and logical pixels.
const (
	// holdTicks is how long a finger rests before it counts as a hold,
	// half a second at the loop's 60 ticks a second. It matches the delay
	// a phone uses before its own press-and-hold menu appears.
	holdTicks = 30
	// holdSlop is how far a finger may wander and still be resting. A
	// board pixel is smaller than a screen pixel once 640 by 480 is scaled
	// down to a phone, so this is roughly a dozen screen pixels, which is
	// ordinary jitter rather than a deliberate drag.
	holdSlop = 20
)

// touch turns one finger into the buttons the game reads.
//
// A phone has no hover and no second button, so the two gestures the game
// needs are recovered from timing. A tap becomes a left click, reported when
// the finger lifts rather than when it lands, because until it lifts there
// is no way to tell a tap from the start of a hold. A finger held still
// becomes the right button, which is how the game reveals a buried card.
//
// The press and the release of a tap are reported on consecutive ticks. A
// dialog presses its button on one and activates it on the next, so a tap
// that reported both at once would arm a button and never fire it.
type touch struct {
	// down is whether a finger is on the board.
	down bool
	// at is where the finger is now, and origin where it landed.
	at, origin image.Point
	// ticks counts how long the finger has been down.
	ticks int
	// wandered records that the finger moved too far to be resting, which
	// rules out a hold for the rest of this press.
	wandered bool
	// holding is whether the hold has been recognised and the right button
	// is being reported.
	holding bool
	// tap counts down the two ticks a lifted tap is reported over: 2 for
	// the press, 1 for the release, 0 for nothing pending.
	tap int
	// tapAt is where the finger lifted, which is where the click lands.
	tapAt image.Point
}

// touchTick is what one tick of a finger amounts to in the game's terms.
type touchTick struct {
	// cursor is where the pointer should be, valid only when moveCursor is
	// set. A finger that is not touching the board leaves the cursor alone,
	// so the board does not react to a pointer that is not there.
	cursor     image.Point
	moveCursor bool

	leftPressed  bool
	leftReleased bool
	rightDown    bool
}

// begin records a finger landing at pt.
func (t *touch) begin(pt image.Point) {
	t.down = true
	t.at, t.origin = pt, pt
	t.ticks = 0
	t.wandered = false
	t.holding = false
}

// move records the finger sliding to pt.
func (t *touch) move(pt image.Point) {
	if !t.down {
		return
	}
	t.at = pt
	if abs(pt.X-t.origin.X) > holdSlop || abs(pt.Y-t.origin.Y) > holdSlop {
		t.wandered = true
	}
}

// end records the finger lifting. A press that never became a hold is a tap,
// wherever it wandered to: the board has nothing to drag and nothing to
// scroll, so a sliding finger is a tap the player was imprecise about.
func (t *touch) end() {
	if !t.down {
		return
	}
	t.down = false
	if !t.holding {
		t.tap, t.tapAt = 2, t.at
	}
	t.holding = false
}

// cancel abandons the press, for a finger the browser takes away.
func (t *touch) cancel() {
	t.down = false
	t.holding = false
	t.tap = 0
}

// tick advances the finger by one tick and reports what the game should see.
func (t *touch) tick() touchTick {
	switch {
	case t.tap == 2:
		t.tap = 1
		return touchTick{cursor: t.tapAt, moveCursor: true, leftPressed: true}
	case t.tap == 1:
		t.tap = 0
		return touchTick{cursor: t.tapAt, moveCursor: true, leftReleased: true}
	case !t.down:
		return touchTick{}
	}

	t.ticks++
	if !t.wandered && t.ticks >= holdTicks {
		t.holding = true
	}
	return touchTick{cursor: t.at, moveCursor: true, rightDown: t.holding}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// boardPoint maps a position within an element box onto the board.
//
// The board keeps its 4 by 3 shape inside whatever box the page gives it,
// scaled to fit, centred across the box and set at its top, so the box is
// usually taller or wider than the picture and the difference is empty
// margin. Mapping against the box rather than the picture inside it puts
// every press in the wrong place on any window that is not itself 4 by 3,
// which a phone never is. It reports false for a press that landed on the
// margin.
func boardPoint(x, y, boxW, boxH float64, board image.Point) (image.Point, bool) {
	scale, offX := fit(board, boxW, boxH)
	if scale == 0 {
		return image.Point{}, false
	}
	pt := image.Pt(int(math.Floor((x-offX)/scale)), int(math.Floor(y/scale)))
	if pt.X < 0 || pt.Y < 0 || pt.X >= board.X || pt.Y >= board.Y {
		return pt, false
	}
	return pt, true
}
