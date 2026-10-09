package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/edition"
)

// drawDialogInto renders d into a recorder at the given edition.
func drawDialogInto(t *testing.T, d *dialog, ed edition.Edition, cursor image.Point) *recorder {
	t.Helper()
	r := &recorder{}
	newTestArt(t).drawDialog(r, d, ed, cursor)
	return r
}

// textAt reports whether any glyph was drawn with its top-left corner at pt.
func (r *recorder) textAt(pt image.Point) bool {
	for _, o := range r.ops {
		if o.op == "BlitTinted" && o.at == pt {
			return true
		}
	}
	return false
}

// filled reports whether rect was filled with c.
func (r *recorder) filled(rect image.Rectangle, c color.Color) bool {
	for _, o := range r.ops {
		if o.op == "FillRect" && o.rect == rect && o.colour == c {
			return true
		}
	}
	return false
}

func TestDrawDialogDrawsTheCaptionAndTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ed   edition.Edition
	}{
		{name: "95", ed: edition.Win95},
		{name: "xp", ed: edition.WinXP},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := newDialog("cellmate", "That move is not allowed.", []string{"OK"}, func(int) {})
			r := drawDialogInto(t, d, tc.ed, image.Point{})

			th := themeFor(tc.ed)
			box := d.rect()
			bar := image.Rect(box.Min.X+frame, box.Min.Y+frame, box.Max.X-frame, box.Min.Y+frame+captionH)
			if !r.filled(bar, th.caption) {
				t.Errorf("no caption bar filled at %v in the %s theme", bar, tc.name)
			}
			// The title sits just inside the caption bar
			if !r.textAt(image.Pt(bar.Min.X+4, bar.Min.Y+2)) {
				t.Error("the title was not drawn in the caption bar")
			}
		})
	}
}

func TestDrawDialogTicksACheckedBox(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		on       bool
		group    int
		wantTick bool
		wantDot  bool
	}{
		{name: "unchecked box", on: false},
		{name: "checked box", on: true, wantTick: true},
		{name: "selected radio", on: true, group: 1, wantDot: true},
		{name: "unselected radio", on: false, group: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := newDialog("Options", "", []string{"OK"}, func(int) {})
			d.checks = []*checkbox{{label: "Quick play", on: tc.on, group: tc.group}}
			r := drawDialogInto(t, d, edition.Win95, image.Point{})

			cr := d.checkRect(0)
			box := image.Rect(cr.Min.X, cr.Min.Y, cr.Min.X+checkBox, cr.Min.Y+checkBox)

			// The tick is drawn as single pixels; the radio dot as one inset block
			pixels := 0
			dot := false
			for _, o := range r.ops {
				if o.op != "FillRect" || o.colour != color.Color(darkBlack) {
					continue
				}
				if o.rect.Dx() == 1 && o.rect.Dy() == 1 && o.rect.In(box) {
					pixels++
				}
				if o.rect == image.Rect(box.Min.X+4, box.Min.Y+4, box.Max.X-4, box.Max.Y-4) {
					dot = true
				}
			}
			if got := pixels > 0; got != tc.wantTick {
				t.Errorf("tick drawn = %v (%d pixels), want %v", got, pixels, tc.wantTick)
			}
			if dot != tc.wantDot {
				t.Errorf("radio dot drawn = %v, want %v", dot, tc.wantDot)
			}
		})
	}
}

func TestTickDrawsTheWindowsCheckMark(t *testing.T) {
	t.Parallel()
	r := &recorder{}
	newTestArt(t).tick(r, image.Pt(0, 0))

	// One filled pixel per x in the shape, and nothing else
	want := 0
	for _, row := range tickShape {
		for _, px := range row {
			if px == 'x' {
				want++
			}
		}
	}
	if got := len(r.ops); got != want {
		t.Fatalf("drew %d pixels, want %d", got, want)
	}
	for _, o := range r.ops {
		if o.rect.Dx() != 1 || o.rect.Dy() != 1 {
			t.Fatalf("drew %v, want a single pixel", o.rect)
		}
	}
}

func TestDrawDialogShowsTheFieldAndItsCaret(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		text      string
		selected  bool
		opened    int
		wantCaret bool
	}{
		{name: "caret shown early in the blink", text: "61", opened: 0, wantCaret: true},
		{name: "caret hidden later in the blink", text: "61", opened: 30},
		{name: "no caret while the text is selected", text: "61", selected: true, opened: 0},
		{name: "caret on an empty field", text: "", opened: 0, wantCaret: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := newDialog("Select Game", "", []string{"OK", "Cancel"}, func(int) {})
			d.field = &textField{text: tc.text, selected: tc.selected}
			d.opened = tc.opened
			r := drawDialogInto(t, d, edition.Win95, image.Point{})

			fr := d.fieldRect()
			caretX := fr.Min.X + 4 + len(tc.text)*glyphW
			caret := image.Rect(caretX, fr.Min.Y+4, caretX+1, fr.Max.Y-4)
			if got := r.filled(caret, darkBlack); got != tc.wantCaret {
				t.Errorf("caret drawn = %v, want %v", got, tc.wantCaret)
			}
		})
	}
}

func TestDrawDialogPressesAButtonUnderTheCursor(t *testing.T) {
	t.Parallel()
	d := newDialog("cellmate", "Resign this game?", []string{"Yes", "No"}, func(int) {})
	d.pressed = 0
	b := d.buttonRect(0)

	// The label shifts a pixel down and right while the button is held
	up := drawDialogInto(t, d, edition.Win95, image.Pt(0, 0))
	down := drawDialogInto(t, d, edition.Win95, b.Min.Add(image.Pt(2, 2)))

	label := "Yes"
	tx := b.Min.X + (b.Dx()-len(label)*glyphW)/2
	ty := b.Min.Y + 5
	if !up.textAt(image.Pt(tx, ty)) {
		t.Error("the released label was not drawn at its resting position")
	}
	if !down.textAt(image.Pt(tx+1, ty+1)) {
		t.Error("the pressed label was not shifted down and right")
	}
}

func TestDrawOpenMenuDrawsNothingWhileClosed(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	if tb.menuOpen >= 0 {
		t.Fatal("a menu is open before anything was clicked")
	}

	r := &recorder{}
	newTestArt(t).drawOpenMenu(r, tb)

	if len(r.ops) != 0 {
		t.Errorf("drew %d calls with no menu open, want none", len(r.ops))
	}
}

func TestDrawOpenMenuDrawsEveryLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ed   edition.Edition
	}{
		{name: "95", ed: edition.Win95},
		{name: "xp", ed: edition.WinXP},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tb, _ := newTestTable(t, edition.WinXP, 1)
			tb.edition = tc.ed
			tb.menuOpen = 0

			r := &recorder{}
			newTestArt(t).drawOpenMenu(r, tb)

			th := themeFor(tc.ed)
			want := 0
			for _, it := range tb.menus()[0].items {
				if it.separator() {
					continue
				}
				n := len(it.label) + len(it.shortcut)
				want += n
				// The 95 theme embosses a disabled item by drawing it a
				// second time, offset, in the highlight colour.
				if !it.canRun() && th.embossed {
					want += n
				}
			}
			glyphs := r.count("BlitTinted")
			if glyphs == 0 {
				t.Fatal("the open menu drew no text at all")
			}
			// Spaces take an advance but draw nothing, so the count lands at
			// or just under the character total.
			if glyphs > want {
				t.Errorf("drew %d glyphs, want no more than the %d characters on show", glyphs, want)
			}
		})
	}
}

func TestDrawOpenMenuEmbossesDisabledItemsOnlyIn95(t *testing.T) {
	t.Parallel()
	// Restart Game is disabled on a fresh deal, which is what gives the 95
	// theme something to emboss.
	count := func(ed edition.Edition) int {
		tb, _ := newTestTable(t, edition.WinXP, 1)
		tb.edition = ed
		tb.menuOpen = 0
		r := &recorder{}
		newTestArt(t).drawOpenMenu(r, tb)
		return r.count("BlitTinted")
	}
	if got95, gotXP := count(edition.Win95), count(edition.WinXP); got95 <= gotXP {
		t.Errorf("95 drew %d glyphs and xp drew %d; 95 should draw more, embossing its disabled items",
			got95, gotXP)
	}
}

func TestDrawOpenMenuHighlightsTheHoveredItem(t *testing.T) {
	t.Parallel()
	tb, _ := newTestTable(t, edition.WinXP, 1)
	tb.menuOpen = 0
	tb.menuHover = -1

	plain := &recorder{}
	newTestArt(t).drawOpenMenu(plain, tb)

	tb.menuHover = 0
	hovered := &recorder{}
	newTestArt(t).drawOpenMenu(hovered, tb)

	th := themeFor(tb.edition)
	ir := menuItemRect(tb.menus(), 0, 0)
	if plain.filled(ir, th.highlight) {
		t.Error("an item was highlighted with nothing hovered")
	}
	if !hovered.filled(ir, th.highlight) {
		t.Errorf("the hovered item at %v was not highlighted", ir)
	}
}
