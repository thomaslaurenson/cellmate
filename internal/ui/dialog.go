package ui

import (
	"image"
	"image/color"
	"strings"
	"unicode"

	"github.com/thomaslaurenson/cellmate/internal/edition"
)

// Message box geometry, following the Windows 95 metrics: 75 by 23 pixel
// buttons, an 18 pixel caption and a 3 pixel frame.
const (
	buttonW     = 75
	buttonH     = 23
	buttonGap   = 6
	captionH    = 18
	frame       = 3
	dialogPad   = 12
	lineH       = 15
	dialogMinW  = 220
	dialogTextY = captionH + frame + dialogPad
	fieldH      = 21
	fieldW      = 90
	checkH      = 20
	checkBox    = 13
)

// Chrome colours shared by the editions.
var (
	hiWhite    = color.RGBA{0xff, 0xff, 0xff, 0xff}
	shadowGrey = color.RGBA{0x80, 0x80, 0x80, 0xff}
	darkBlack  = color.RGBA{0x00, 0x00, 0x00, 0xff}
)

// textField is a one-line box for typing a number. Its starting text is
// selected, as Windows selects the contents of a box when a dialog opens,
// so typing replaces it and Backspace clears it.
type textField struct {
	text     string
	maxLen   int
	selected bool
}

func newTextField(text string, maxLen int) *textField {
	return &textField{text: text, maxLen: maxLen, selected: true}
}

// checkbox is an on-off choice. Boxes sharing a non-zero group behave as
// radio buttons: choosing one clears the others, and a chosen one cannot be
// cleared by clicking it again.
type checkbox struct {
	label string
	on    bool
	group int
}

// dialog is a modal message box. choose runs with the index of the button
// picked; Enter picks the first button and Escape the last, as Windows did
// for a default button and a cancel button.
type dialog struct {
	title   string
	lines   []string
	field   *textField
	checks  []*checkbox
	buttons []string
	// hotkeys holds a letter per button that picks it when typed, as the
	// underlined letters did in Windows. Zero means none.
	hotkeys []rune
	choose  func(int)
	pressed int
	// opened counts ticks since the dialog appeared, to blink the caret.
	opened int
}

func newDialog(title, message string, buttons []string, choose func(int)) *dialog {
	var lines []string
	if message != "" {
		lines = strings.Split(message, "\n")
	}
	return &dialog{
		title:   title,
		lines:   lines,
		buttons: buttons,
		hotkeys: defaultHotkeys(buttons),
		choose:  choose,
		pressed: -1,
	}
}

// defaultHotkeys gives each button its first letter, where no other button
// starts with the same one.
func defaultHotkeys(buttons []string) []rune {
	keys := make([]rune, len(buttons))
	count := map[rune]int{}
	for _, b := range buttons {
		count[unicode.ToLower(rune(b[0]))]++
	}
	for i, b := range buttons {
		if r := unicode.ToLower(rune(b[0])); count[r] == 1 {
			keys[i] = r
		}
	}
	return keys
}

// buttonW returns the width shared by the dialog's buttons: the standard
// width, widened to fit the longest label.
func (d *dialog) buttonW() int {
	w := buttonW
	for _, b := range d.buttons {
		w = max(w, len(b)*glyphW+16)
	}
	return w
}

// fieldY and checksY are the offsets from the top of the box to the text
// field and the first checkbox.
func (d *dialog) fieldY() int { return dialogTextY + len(d.lines)*lineH }

func (d *dialog) checksY() int {
	y := d.fieldY()
	if d.field != nil {
		y += fieldH + dialogPad
	}
	return y
}

// rect returns the box, centred on the board.
func (d *dialog) rect() image.Rectangle {
	inner := 2 * (dialogPad + frame)
	w := dialogMinW
	for _, l := range append([]string{d.title}, d.lines...) {
		w = max(w, len(l)*glyphW+inner)
	}
	for _, c := range d.checks {
		w = max(w, checkBox+8+len(c.label)*glyphW+inner)
	}
	w = max(w, len(d.buttons)*(d.buttonW()+buttonGap)-buttonGap+inner)
	h := d.checksY() + len(d.checks)*checkH + dialogPad + buttonH + dialogPad + frame
	x := (screenW - w) / 2
	y := (screenH - h) / 2
	return image.Rect(x, y, x+w, y+h)
}

func (d *dialog) fieldRect() image.Rectangle {
	r := d.rect()
	x, y := r.Min.X+frame+dialogPad, r.Min.Y+d.fieldY()
	return image.Rect(x, y, x+fieldW, y+fieldH)
}

// checkRect returns checkbox i with its label, the whole of which is
// clickable.
func (d *dialog) checkRect(i int) image.Rectangle {
	r := d.rect()
	x, y := r.Min.X+frame+dialogPad, r.Min.Y+d.checksY()+i*checkH
	return image.Rect(x, y, x+checkBox+8+len(d.checks[i].label)*glyphW, y+checkBox+2)
}

// buttonRect returns button i, the row centred along the bottom of the box.
func (d *dialog) buttonRect(i int) image.Rectangle {
	r := d.rect()
	bw := d.buttonW()
	row := len(d.buttons)*(bw+buttonGap) - buttonGap
	x := r.Min.X + (r.Dx()-row)/2 + i*(bw+buttonGap)
	y := r.Max.Y - frame - dialogPad - buttonH
	return image.Rect(x, y, x+bw, y+buttonH)
}

// buttonAt returns the button under pt, or -1. The buttons sit in the
// dialog's padding with a gap between them, so a finger's margin has
// somewhere to go without reaching its neighbour.
func (d *dialog) buttonAt(pt image.Point, margin int) int {
	for i := range d.buttons {
		if pt.In(grow(d.buttonRect(i), min(margin, buttonGap/2))) {
			return i
		}
	}
	return -1
}

// checkHitRect returns what answers a tap for checkbox i. The drawn row is
// shorter than the space between rows, so filling that space costs nothing
// and gives a finger the whole line.
func (d *dialog) checkHitRect(i int) image.Rectangle {
	r := d.checkRect(i)
	r.Max.Y = r.Min.Y + checkH
	return r
}

func (d *dialog) toggle(i int) {
	c := d.checks[i]
	if c.group == 0 {
		c.on = !c.on
		return
	}
	for _, o := range d.checks {
		if o.group == c.group {
			o.on = o == c
		}
	}
}

// update handles one tick of input and reports whether the dialog closed.
// A button fires on release over the button it was pressed on, so dragging
// off it cancels, as it does in Windows.
func (d *dialog) update(in input) bool {
	d.opened++
	if f := d.field; f != nil {
		for _, r := range in.chars {
			// A minus sign is taken first and nowhere else, since the two
			// impossible deals are numbered -1 and -2; anything else that
			// is not a digit is dropped.
			digit := r >= '0' && r <= '9'
			minus := r == '-' && (f.selected || f.text == "")
			if !digit && !minus {
				continue
			}
			if f.selected {
				f.text, f.selected = "", false
			}
			if len(f.text) < f.maxLen {
				f.text += string(r)
			}
		}
		if in.backspace {
			if f.selected {
				f.text, f.selected = "", false
			} else if f.text != "" {
				f.text = f.text[:len(f.text)-1]
			}
		}
	}

	pick := -1
	for _, r := range in.chars {
		for i, k := range d.hotkeys {
			if k != 0 && unicode.ToLower(r) == k {
				pick = i
			}
		}
	}
	switch {
	case pick >= 0:
	case in.enter:
		pick = 0
	case in.escape:
		pick = len(d.buttons) - 1
	case in.leftPressed:
		d.pressed = d.buttonAt(in.cursor, in.margin())
		for i := range d.checks {
			if in.cursor.In(d.checkHitRect(i)) {
				d.toggle(i)
			}
		}
	case in.leftReleased:
		if d.pressed >= 0 && d.buttonAt(in.cursor, in.margin()) == d.pressed {
			pick = d.pressed
		}
		d.pressed = -1
	}
	if pick < 0 {
		return false
	}
	d.choose(pick)
	return true
}

func (a *art) drawDialog(dst surface, d *dialog, ed edition.Edition, cursor image.Point) {
	th := themeFor(ed)
	r := d.rect()

	a.fillRect(dst, r, th.face)
	a.bevel(dst, r, th.face, darkBlack)
	a.bevel(dst, r.Inset(1), hiWhite, shadowGrey)

	bar := image.Rect(r.Min.X+frame, r.Min.Y+frame, r.Max.X-frame, r.Min.Y+frame+captionH)
	a.fillRect(dst, bar, th.caption)
	a.text(dst, d.title, bar.Min.X+4, bar.Min.Y+2, hiWhite)

	for i, l := range d.lines {
		a.text(dst, l, r.Min.X+frame+dialogPad, r.Min.Y+dialogTextY+i*lineH, darkBlack)
	}

	if f := d.field; f != nil {
		fr := d.fieldRect()
		a.sunken(dst, fr, hiWhite)
		if f.selected && f.text != "" {
			sel := image.Rect(fr.Min.X+3, fr.Min.Y+3, fr.Min.X+5+len(f.text)*glyphW, fr.Max.Y-3)
			a.fillRect(dst, sel, th.highlight)
			a.text(dst, f.text, fr.Min.X+4, fr.Min.Y+4, hiWhite)
		} else {
			a.text(dst, f.text, fr.Min.X+4, fr.Min.Y+4, darkBlack)
		}
		if !f.selected && d.opened/30%2 == 0 {
			x := fr.Min.X + 4 + len(f.text)*glyphW
			a.fillRect(dst, image.Rect(x, fr.Min.Y+4, x+1, fr.Max.Y-4), darkBlack)
		}
	}

	for i, c := range d.checks {
		cr := d.checkRect(i)
		box := image.Rect(cr.Min.X, cr.Min.Y, cr.Min.X+checkBox, cr.Min.Y+checkBox)
		a.sunken(dst, box, hiWhite)
		if c.on {
			if c.group != 0 {
				a.fillRect(dst, image.Rect(box.Min.X+4, box.Min.Y+4, box.Max.X-4, box.Max.Y-4), darkBlack)
			} else {
				a.tick(dst, box.Min)
			}
		}
		a.text(dst, c.label, box.Max.X+6, cr.Min.Y, darkBlack)
	}

	for i, label := range d.buttons {
		b := d.buttonRect(i)
		down := d.pressed == i && cursor.In(b)
		a.fillRect(dst, b, th.face)
		if down {
			a.bevel(dst, b, darkBlack, hiWhite)
			a.bevel(dst, b.Inset(1), shadowGrey, th.face)
		} else {
			a.bevel(dst, b, hiWhite, darkBlack)
			a.bevel(dst, b.Inset(1), th.face, shadowGrey)
		}
		if i == 0 {
			a.bevel(dst, b.Inset(-1), darkBlack, darkBlack)
		}
		tx := b.Min.X + (b.Dx()-len(label)*glyphW)/2
		ty := b.Min.Y + 5
		if down {
			tx, ty = tx+1, ty+1
		}
		a.text(dst, label, tx, ty, darkBlack)
	}
}

// sunken draws a white well with the two-pixel inset border Windows used
// for text boxes and checkboxes.
func (a *art) sunken(dst surface, r image.Rectangle, fill color.Color) {
	a.fillRect(dst, r, fill)
	a.bevel(dst, r, shadowGrey, hiWhite)
	a.bevel(dst, r.Inset(1), darkBlack, color.RGBA{0xdf, 0xdf, 0xdf, 0xff})
}

// tickShape is the 7 by 7 check mark from the Windows 95 checkbox, one
// string per row with x for a set pixel.
var tickShape = []string{
	"......x",
	".....xx",
	"x...xxx",
	"xx.xxx.",
	"xxxxx..",
	".xxx...",
	"..x....",
}

func (a *art) tick(dst surface, at image.Point) {
	for y, row := range tickShape {
		for x, px := range row {
			if px == 'x' {
				p := at.Add(image.Pt(3+x, 3+y))
				a.fillRect(dst, image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}, darkBlack)
			}
		}
	}
}
