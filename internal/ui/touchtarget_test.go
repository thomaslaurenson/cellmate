package ui

import (
	"image"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/edition"
)

func TestMarginIsOnlyGivenToAFinger(t *testing.T) {
	t.Parallel()
	if got := (input{}).margin(); got != 0 {
		t.Errorf("a mouse was given %d pixels of margin, want none", got)
	}
	if got := (input{touch: true}).margin(); got != touchMargin {
		t.Errorf("a finger was given %d pixels of margin, want %d", got, touchMargin)
	}
}

func TestDialogButtonAcceptsANearMissFromAFinger(t *testing.T) {
	t.Parallel()
	d := newDialog("cellmate", "Resign this game?", []string{"Yes", "No"}, func(int) {})
	b := d.buttonRect(0)

	tests := []struct {
		name   string
		pt     image.Point
		margin int
		want   int
	}{
		{name: "dead centre with a mouse", pt: centreOf(b), want: 0},
		{name: "dead centre with a finger", pt: centreOf(b), margin: touchMargin, want: 0},
		{name: "just above, with a mouse", pt: image.Pt(centreOf(b).X, b.Min.Y-2), want: -1},
		{name: "just above, with a finger", pt: image.Pt(centreOf(b).X, b.Min.Y-2), margin: touchMargin, want: 0},
		{name: "just below, with a finger", pt: image.Pt(centreOf(b).X, b.Max.Y+2), margin: touchMargin, want: 0},
		{name: "well away, even with a finger", pt: image.Pt(b.Min.X-60, b.Min.Y-60), margin: touchMargin, want: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := d.buttonAt(tc.pt, tc.margin); got != tc.want {
				t.Errorf("buttonAt(%v, %d) = %d, want %d", tc.pt, tc.margin, got, tc.want)
			}
		})
	}
}

func TestDialogButtonsNeverStealFromEachOther(t *testing.T) {
	t.Parallel()
	// The margin is capped at half the gap between buttons, so the widened
	// areas meet but never overlap. A tap in the gap must not be ambiguous.
	d := newDialog("cellmate", "Resign this game?", []string{"Yes", "No"}, func(int) {})
	first, second := d.buttonRect(0), d.buttonRect(1)

	for x := first.Max.X; x < second.Min.X; x++ {
		pt := image.Pt(x, centreOf(first).Y)
		got := d.buttonAt(pt, touchMargin)
		if got != 0 && got != 1 && got != -1 {
			t.Fatalf("buttonAt(%v) = %d, want one of the two buttons or none", pt, got)
		}
	}
	// Each button still answers for itself at its own centre
	if d.buttonAt(centreOf(first), touchMargin) != 0 {
		t.Error("the first button stopped answering at its centre")
	}
	if d.buttonAt(centreOf(second), touchMargin) != 1 {
		t.Error("the second button stopped answering at its centre")
	}
}

func TestCheckboxAnswersForItsWholeRow(t *testing.T) {
	t.Parallel()
	d := newDialog("Options", "", []string{"OK", "Cancel"}, func(int) {})
	d.checks = []*checkbox{
		{label: "Display messages on illegal moves", on: true},
		{label: "Quick play (no animation)"},
	}

	drawn, hit := d.checkRect(0), d.checkHitRect(0)
	if hit.Dy() <= drawn.Dy() {
		t.Fatalf("the hit row is %d pixels tall against the drawn %d, want taller",
			hit.Dy(), drawn.Dy())
	}
	if hit.Dy() != checkH {
		t.Errorf("the hit row is %d pixels tall, want the %d between rows", hit.Dy(), checkH)
	}
	// Rows must tile without overlapping, or a tap would toggle two options
	if next := d.checkHitRect(1); hit.Overlaps(next) {
		t.Errorf("row 0 at %v overlaps row 1 at %v", hit, next)
	}
	// The gap the drawn box leaves below itself answers too
	below := image.Pt(drawn.Min.X+2, drawn.Max.Y+1)
	if !below.In(hit) {
		t.Errorf("a tap at %v just under the box still misses the row %v", below, hit)
	}
}

func TestMenuTitleGrowsDownwardsOnly(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	menus := tb.menus()
	r := menuTitleRect(menus, 0)

	tests := []struct {
		name   string
		pt     image.Point
		margin int
		want   int
	}{
		{name: "on the title with a mouse", pt: centreOf(r), want: 0},
		{name: "below the bar with a mouse", pt: image.Pt(centreOf(r).X, menuH+4), want: -1},
		{name: "below the bar with a finger", pt: image.Pt(centreOf(r).X, menuH+4), margin: touchMargin, want: 0},
		{
			name: "a finger does not reach sideways into the next title",
			// Growing sideways would take taps from Help, which sits
			// immediately to the right with no gap.
			pt: image.Pt(r.Max.X+2, centreOf(r).Y), margin: touchMargin, want: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := menuTitleAt(menus, tc.pt, tc.margin); got != tc.want {
				t.Errorf("menuTitleAt(%v, %d) = %d, want %d", tc.pt, tc.margin, got, tc.want)
			}
		})
	}
}

func TestGrowLeavesAMouseAlone(t *testing.T) {
	t.Parallel()
	r := image.Rect(10, 20, 30, 40)
	if got := grow(r, 0); got != r {
		t.Errorf("grow with no margin gave %v, want %v", got, r)
	}
	if got := growDown(r, 0); got != r {
		t.Errorf("growDown with no margin gave %v, want %v", got, r)
	}
	if got := growDown(r, 5); got != image.Rect(10, 20, 30, 45) {
		t.Errorf("growDown grew to %v, want only the bottom edge to move", got)
	}
}

// centreOf returns the middle of r.
func centreOf(r image.Rectangle) image.Point {
	return image.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2)
}
