package gfx

import (
	"image"
	"testing"
)

// run advances tr for n ticks and returns every tick's result.
func run(tr *touch, n int) []touchTick {
	out := make([]touchTick, n)
	for i := range n {
		out[i] = tr.tick()
	}
	return out
}

// presses counts the ticks that reported a left press.
func presses(ts []touchTick) int {
	n := 0
	for _, t := range ts {
		if t.leftPressed {
			n++
		}
	}
	return n
}

func TestTouchIdleReportsNothing(t *testing.T) {
	t.Parallel()
	var tr touch

	for i, got := range run(&tr, 5) {
		if got != (touchTick{}) {
			t.Fatalf("tick %d of an untouched board reported %+v", i, got)
		}
	}
}

func TestTouchTapClicksWhereTheFingerLifts(t *testing.T) {
	t.Parallel()
	var tr touch
	tr.begin(image.Pt(100, 200))
	run(&tr, 3) // A brief press, nowhere near a hold
	tr.end()

	got := run(&tr, 3)

	if !got[0].leftPressed || got[0].leftReleased {
		t.Errorf("first tick after the lift = %+v, want the press alone", got[0])
	}
	if !got[1].leftReleased || got[1].leftPressed {
		t.Errorf("second tick = %+v, want the release alone", got[1])
	}
	if got[2] != (touchTick{}) {
		t.Errorf("third tick = %+v, want nothing left over", got[2])
	}
	for i, tick := range got[:2] {
		if !tick.moveCursor || tick.cursor != image.Pt(100, 200) {
			t.Errorf("tick %d put the cursor at %v, want the tap point", i, tick.cursor)
		}
	}
}

func TestTouchTapReportsNothingWhileTheFingerIsDown(t *testing.T) {
	t.Parallel()
	var tr touch
	tr.begin(image.Pt(10, 10))

	// Until the finger lifts there is no way to tell a tap from the start
	// of a hold, so nothing is clicked yet.
	for i, got := range run(&tr, holdTicks-1) {
		if got.leftPressed || got.leftReleased {
			t.Fatalf("tick %d clicked while the finger was still down: %+v", i, got)
		}
		if got.rightDown {
			t.Fatalf("tick %d reported a hold before the delay was up", i)
		}
	}
}

func TestTouchHoldBecomesTheRightButton(t *testing.T) {
	t.Parallel()
	var tr touch
	tr.begin(image.Pt(50, 60))

	before := run(&tr, holdTicks-1)
	if before[len(before)-1].rightDown {
		t.Fatal("the hold fired a tick early")
	}
	after := run(&tr, 5)
	for i, got := range after {
		if !got.rightDown {
			t.Fatalf("tick %d after the delay = %+v, want the right button held", i, got)
		}
		if got.cursor != image.Pt(50, 60) {
			t.Errorf("tick %d put the cursor at %v, want where the finger rests", i, got.cursor)
		}
	}
}

func TestTouchHoldSuppressesTheTap(t *testing.T) {
	t.Parallel()
	var tr touch
	tr.begin(image.Pt(50, 60))
	run(&tr, holdTicks+2)
	tr.end()

	got := run(&tr, 3)
	if n := presses(got); n != 0 {
		t.Errorf("lifting after a hold clicked %d times, want none", n)
	}
	for i, tick := range got {
		if tick.rightDown {
			t.Errorf("tick %d still held the right button after the lift", i)
		}
	}
}

func TestTouchWanderingRulesOutTheHold(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		to       image.Point
		wantHold bool
	}{
		{name: "resting still", to: image.Pt(50, 60), wantHold: true},
		{name: "jitter within the slop", to: image.Pt(50+holdSlop, 60), wantHold: true},
		{name: "wandered along x", to: image.Pt(50+holdSlop+1, 60)},
		{name: "wandered along y", to: image.Pt(50, 60+holdSlop+1)},
		{name: "wandered back again", to: image.Pt(50, 60)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var tr touch
			tr.begin(image.Pt(50, 60))
			run(&tr, 5)
			if tc.name == "wandered back again" {
				// Straying once rules out the hold for the rest of the
				// press, even if the finger comes back.
				tr.move(image.Pt(50+holdSlop+40, 60))
			}
			tr.move(tc.to)

			got := run(&tr, holdTicks+5)
			held := got[len(got)-1].rightDown
			if held != tc.wantHold {
				t.Errorf("right button held = %v, want %v", held, tc.wantHold)
			}
		})
	}
}

func TestTouchWanderedFingerStillTaps(t *testing.T) {
	t.Parallel()
	// The board has nothing to drag and nothing to scroll, so a sliding
	// finger is a tap the player was imprecise about.
	var tr touch
	tr.begin(image.Pt(10, 10))
	run(&tr, 3)
	tr.move(image.Pt(200, 300))
	tr.end()

	got := run(&tr, 2)
	if !got[0].leftPressed {
		t.Fatal("a finger that slid before lifting did not click")
	}
	if got[0].cursor != image.Pt(200, 300) {
		t.Errorf("clicked at %v, want where the finger lifted", got[0].cursor)
	}
}

func TestTouchCancelDropsEverything(t *testing.T) {
	t.Parallel()
	var tr touch
	tr.begin(image.Pt(50, 60))
	run(&tr, holdTicks+2)
	tr.cancel()

	for i, got := range run(&tr, 3) {
		if got != (touchTick{}) {
			t.Errorf("tick %d after the cancel reported %+v, want nothing", i, got)
		}
	}
}

func TestTouchTwoTapsClickTwice(t *testing.T) {
	t.Parallel()
	// Double tap sends a card to a free cell, so two taps in quick
	// succession must arrive as two separate clicks.
	var tr touch
	var ticks []touchTick
	for range 2 {
		tr.begin(image.Pt(80, 90))
		ticks = append(ticks, run(&tr, 2)...)
		tr.end()
		ticks = append(ticks, run(&tr, 2)...)
	}

	if n := presses(ticks); n != 2 {
		t.Errorf("two taps produced %d clicks, want 2", n)
	}
}

func TestTouchEndWithoutBeginIsIgnored(t *testing.T) {
	t.Parallel()
	var tr touch
	tr.end()
	tr.move(image.Pt(5, 5))

	for i, got := range run(&tr, 3) {
		if got != (touchTick{}) {
			t.Errorf("tick %d reported %+v for a finger that never landed", i, got)
		}
	}
}

func TestBoardPointMapsThroughTheLetterbox(t *testing.T) {
	t.Parallel()
	board := image.Pt(640, 480)
	tests := []struct {
		name        string
		x, y        float64
		boxW, boxH  float64
		want        image.Point
		wantOnBoard bool
	}{
		{
			name: "a box of the board's own shape maps one to one",
			x:    320, y: 240, boxW: 640, boxH: 480,
			want: image.Pt(320, 240), wantOnBoard: true,
		},
		{
			name: "a box of the board's shape, scaled down",
			x:    160, y: 120, boxW: 320, boxH: 240,
			want: image.Pt(320, 240), wantOnBoard: true,
		},
		{
			name: "a tall phone box sets the board at the top",
			// 390 wide scales the board to 390 by 292.5, and the spare
			// height of a 664 tall box is all below it.
			x: 195, y: 146.25, boxW: 390, boxH: 664,
			want: image.Pt(320, 240), wantOnBoard: true,
		},
		{
			name: "the top left of a tall box is the board's top left",
			x:    0, y: 0, boxW: 390, boxH: 664,
			want: image.Pt(0, 0), wantOnBoard: true,
		},
		{
			name: "the board's bottom row is on it",
			x:    195, y: 292.4, boxW: 390, boxH: 664,
			want: image.Pt(320, 479), wantOnBoard: true,
		},
		{
			name: "the margin below the board is not on it",
			x:    195, y: 292.5, boxW: 390, boxH: 664,
			wantOnBoard: false,
		},
		{
			name: "a wide box centres the board horizontally",
			// 480 tall scales the board to 640 by 480 in a 1000 wide box,
			// leaving 180 of margin each side.
			x: 180, y: 0, boxW: 1000, boxH: 480,
			want: image.Pt(0, 0), wantOnBoard: true,
		},
		{
			name: "the margin beside a wide box is not on the board",
			x:    40, y: 240, boxW: 1000, boxH: 480,
			wantOnBoard: false,
		},
		{
			name: "a box with no size maps nothing",
			x:    10, y: 10, boxW: 0, boxH: 0,
			wantOnBoard: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, on := boardPoint(tc.x, tc.y, tc.boxW, tc.boxH, board)
			if on != tc.wantOnBoard {
				t.Fatalf("on board = %v, want %v (point %v)", on, tc.wantOnBoard, got)
			}
			if tc.wantOnBoard && got != tc.want {
				t.Errorf("mapped to %v, want %v", got, tc.want)
			}
		})
	}
}
