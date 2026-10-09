package gfx

import (
	"image"
	"testing"
)

// TestFitPlacesTheBoardInTheBox pins where the board goes: centred across
// the box, and at its top. A window's menu bar belongs at the top of the
// window, so a tall box has its whole margin below the board rather than
// half of it above.
func TestFitPlacesTheBoardInTheBox(t *testing.T) {
	t.Parallel()
	board := image.Pt(640, 480)
	tests := []struct {
		name       string
		boxW, boxH float64
		wantScale  float64
		wantOffX   float64
	}{
		{name: "the board's own size", boxW: 640, boxH: 480, wantScale: 1},
		{name: "twice the size", boxW: 1280, boxH: 960, wantScale: 2},
		{name: "a wide box leaves a margin either side", boxW: 1000, boxH: 480, wantScale: 1, wantOffX: 180},
		{name: "a tall box leaves its margin below", boxW: 640, boxH: 800, wantScale: 1},
		{name: "a phone on its side scales down", boxW: 320, boxH: 240, wantScale: 0.5},
		{name: "no box fits nothing", boxW: 0, boxH: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scale, offX := fit(board, tc.boxW, tc.boxH)
			if scale != tc.wantScale || offX != tc.wantOffX {
				t.Errorf("fit = %v at %v in, want %v at %v in", scale, offX, tc.wantScale, tc.wantOffX)
			}
		})
	}
}

// TestViewRectEdgesMeet guards the seams: at a scale that is not a whole
// number, two rectangles that share an edge on the board must share one on
// the surface, or the table shows a line of background between every pair
// of cells.
func TestViewRectEdgesMeet(t *testing.T) {
	t.Parallel()
	v := view{board: image.Pt(640, 480)}
	v.resize(960, 720)
	if v.scale != 1.5 {
		t.Fatalf("scale = %v, want 1.5", v.scale)
	}
	left := v.rect(image.Rect(0, 0, 71, 96))
	right := v.rect(image.Rect(71, 0, 142, 96))
	if left.Max.X != right.Min.X {
		t.Errorf("the left rectangle ends at %d and the right begins at %d", left.Max.X, right.Min.X)
	}
	if want := image.Rect(0, 0, 107, 144); left != want {
		t.Errorf("left = %v, want %v", left, want)
	}
}

// TestViewKeepsOnePixelLinesWhole guards the bevels. The board draws its
// sunken frames as rectangles one logical pixel thick, and at a scale that
// is not a whole number each must still come out a whole number of screen
// pixels and never none at all, or a cell's outline thins and vanishes
// along its length. Consecutive lines must also tile, so a frame has
// neither a gap nor a doubled edge in it.
func TestViewKeepsOnePixelLinesWhole(t *testing.T) {
	t.Parallel()
	// The sizes a laptop actually lands on: a MacBook's 1440 by 820 at two
	// device pixels to one, Windows at 125% and 150%, and a phone on its
	// side at three.
	boxes := []image.Point{{X: 2880, Y: 1640}, {X: 1920, Y: 913}, {X: 960, Y: 720}, {X: 2532, Y: 1170}}
	for _, box := range boxes {
		v := view{board: image.Pt(640, 480)}
		v.resize(box.X, box.Y)
		prev := v.rect(image.Rect(0, 0, 640, 1))
		if prev.Dy() < 1 {
			t.Fatalf("at %v the first line is %d pixels tall", box, prev.Dy())
		}
		for y := 1; y < 480; y++ {
			line := v.rect(image.Rect(0, y, 640, y+1))
			if line.Dy() < 1 {
				t.Fatalf("at %v the line at y=%d is %d pixels tall", box, y, line.Dy())
			}
			if line.Min.Y != prev.Max.Y {
				t.Fatalf("at %v the line at y=%d starts at %d, want %d",
					box, y, line.Min.Y, prev.Max.Y)
			}
			prev = line
		}
	}
}

func TestViewSettlesOnceTheScaleHoldsStill(t *testing.T) {
	t.Parallel()
	v := view{board: image.Pt(640, 480)}
	v.resize(640, 480)
	if v.settled != 1 {
		t.Fatalf("the first frame settled at %v, want 1", v.settled)
	}

	v.resize(1280, 960)
	if v.scale != 2 || v.settled != 1 {
		t.Fatalf("after a resize scale = %v and settled = %v, want 2 and 1", v.scale, v.settled)
	}
	for range settleFrames - 1 {
		v.resize(1280, 960)
	}
	if v.settled != 1 {
		t.Fatalf("settled at %v before the scale had held still", v.settled)
	}
	v.resize(1280, 960)
	if v.settled != 2 {
		t.Errorf("settled = %v after the scale held still, want 2", v.settled)
	}

	// A resize part way through the wait starts it again
	v.resize(1920, 1440)
	for range settleFrames / 2 {
		v.resize(1920, 1440)
	}
	v.resize(640, 480)
	for range settleFrames - 1 {
		v.resize(640, 480)
	}
	if v.settled != 2 {
		t.Errorf("settled = %v after an interrupted wait, want still 2", v.settled)
	}
}

func TestViewMapsPointsBothWays(t *testing.T) {
	t.Parallel()
	v := view{board: image.Pt(640, 480)}
	// Twice the size, with a margin of 40 below the board
	v.resize(1280, 1000)

	if got := v.point(image.Pt(320, 240)); got != image.Pt(640, 480) {
		t.Errorf("point = %v, want (640, 480)", got)
	}
	if got := v.logical(640, 480); got != image.Pt(320, 240) {
		t.Errorf("logical = %v, want (320, 240)", got)
	}
	if got := v.logical(0, 0); got != (image.Point{}) {
		t.Errorf("the top left corner mapped to %v, want the origin", got)
	}
	if got := v.logical(0, 999); got != image.Pt(0, 499) {
		t.Errorf("the margin mapped to %v, want (0, 499), which is off the board", got)
	}

	var blank view
	if got := blank.logical(100, 100); got != (image.Point{}) {
		t.Errorf("a view with no size mapped to %v, want the origin", got)
	}
}

func TestRasterSizeRoundsUp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		size  image.Point
		scale float64
		want  image.Point
	}{
		{name: "one to one", size: image.Pt(71, 96), scale: 1, want: image.Pt(71, 96)},
		{name: "a fraction rounds up", size: image.Pt(71, 96), scale: 1.5, want: image.Pt(107, 144)},
		{name: "never empty", size: image.Pt(71, 96), scale: 0.001, want: image.Pt(1, 1)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := RasterSize(tc.size, tc.scale); got != tc.want {
				t.Errorf("rasterSize = %v, want %v", got, tc.want)
			}
		})
	}
}
