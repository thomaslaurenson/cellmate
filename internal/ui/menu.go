package ui

import (
	"image"
	"strconv"

	"github.com/thomaslaurenson/cellmate/internal/edition"
)

// Menu geometry, from the Windows 95 metrics.
const (
	menuItemH  = 18
	menuSepH   = 8
	menuFrame  = 3
	menuTextX  = 20
	menuMinW   = 150
	menuTitleX = 2
)

// menuItem is one entry in a dropdown. A separator has no label.
type menuItem struct {
	label    string
	shortcut string
	run      func()
	// enabled reports whether the item can be chosen; nil means always.
	enabled func() bool
}

func (m menuItem) separator() bool { return m.label == "" }

func (m menuItem) canRun() bool {
	return !m.separator() && (m.enabled == nil || m.enabled())
}

type menu struct {
	title string
	items []menuItem
}

var separator = menuItem{}

// menus builds the menu bar. It is rebuilt when needed rather than stored,
// because what it offers depends on the edition and on the game.
func (t *table) menus() []menu {
	gameMenu := []menuItem{
		{label: "New Game", shortcut: "F2", run: t.newGame},
		{label: "Select Game...", shortcut: "F3", run: t.selectGame},
		{label: "Restart Game", run: t.restartGame, enabled: t.g.CanUndo},
		separator,
		{label: "Statistics...", shortcut: "F4", run: t.showStats},
		{label: "Options...", shortcut: "F5", run: t.showOptions},
	}
	if t.edition.HasUndo() {
		gameMenu = append(gameMenu, separator,
			menuItem{label: "Undo", shortcut: "F10", run: t.undo, enabled: t.g.CanUndo})
	}
	if t.canExit {
		gameMenu = append(gameMenu, separator, menuItem{label: "Exit", run: t.exitGame})
	}
	return []menu{
		{title: "Game", items: gameMenu},
		{title: "Help", items: []menuItem{
			{label: "How to Play", shortcut: "F1", run: t.showHelp},
			separator,
			{label: "About cellmate...", run: t.showAbout},
		}},
	}
}

// menuTitleRect returns the clickable title of menu i in the menu bar.
func menuTitleRect(menus []menu, i int) image.Rectangle {
	x := menuTitleX
	for j := range i {
		x += len(menus[j].title)*glyphW + 14
	}
	return image.Rect(x, 1, x+len(menus[i].title)*glyphW+14, menuH-1)
}

// menuRect returns the open dropdown of menu i.
func menuRect(menus []menu, i int) image.Rectangle {
	w := menuMinW
	h := 2 * menuFrame
	for _, it := range menus[i].items {
		w = max(w, menuTextX+(len(it.label)+len(it.shortcut)+3)*glyphW+menuTextX)
		if it.separator() {
			h += menuSepH
		} else {
			h += menuItemH
		}
	}
	x := menuTitleRect(menus, i).Min.X
	return image.Rect(x, menuH, x+w, menuH+h)
}

// menuItemRect returns item j of the open dropdown of menu i.
func menuItemRect(menus []menu, i, j int) image.Rectangle {
	r := menuRect(menus, i)
	y := r.Min.Y + menuFrame
	for k := range j {
		if menus[i].items[k].separator() {
			y += menuSepH
		} else {
			y += menuItemH
		}
	}
	h := menuItemH
	if menus[i].items[j].separator() {
		h = menuSepH
	}
	return image.Rect(r.Min.X+menuFrame, y, r.Max.X-menuFrame, y+h)
}

// menuTitleAt returns the menu title under pt, or -1. A finger is allowed
// margin below the bar, which costs the two leftmost free cells a sliver of
// their ninety-six pixel height.
func menuTitleAt(menus []menu, pt image.Point, margin int) int {
	for i := range menus {
		if pt.In(growDown(menuTitleRect(menus, i), margin)) {
			return i
		}
	}
	return -1
}

// menuItemAt returns the item under pt in the open dropdown, or -1.
func menuItemAt(menus []menu, i int, pt image.Point) int {
	for j := range menus[i].items {
		if pt.In(menuItemRect(menus, i, j)) {
			return j
		}
	}
	return -1
}

// updateMenus handles input for the menu bar and reports whether it used
// the tick's input, so the board does not see a click meant for a menu.
func (t *table) updateMenus(in input) bool {
	menus := t.menus()
	if t.menuOpen < 0 {
		if in.leftPressed {
			if i := menuTitleAt(menus, in.cursor, in.margin()); i >= 0 {
				t.menuOpen, t.menuHover = i, -1
				t.selected = nil
				return true
			}
		}
		return false
	}

	// While a menu is open, pointing at another title opens that one
	// instead, as sliding along a Windows menu bar does.
	if i := menuTitleAt(menus, in.cursor, in.margin()); i >= 0 && i != t.menuOpen {
		t.menuOpen, t.menuHover = i, -1
	}
	t.menuHover = -1
	if j := menuItemAt(menus, t.menuOpen, in.cursor); j >= 0 && menus[t.menuOpen].items[j].canRun() {
		t.menuHover = j
	}

	switch {
	case in.escape:
		t.menuOpen = -1
	case in.leftPressed:
		open := t.menuOpen
		t.menuOpen = -1
		if menuTitleAt(menus, in.cursor, in.margin()) >= 0 {
			return true
		}
		if j := menuItemAt(menus, open, in.cursor); j >= 0 && menus[open].items[j].canRun() {
			menus[open].items[j].run()
		}
	}
	return true
}

func (a *art) drawMenuBar(dst surface, t *table) {
	th := themeFor(t.edition)
	a.fillRect(dst, image.Rect(0, 0, screenW, menuH), th.face)
	a.fillRect(dst, image.Rect(0, menuH-1, screenW, menuH), shadowGrey)

	menus := t.menus()
	for i, m := range menus {
		r := menuTitleRect(menus, i)
		ink := darkBlack
		if i == t.menuOpen {
			a.fillRect(dst, r, th.highlight)
			ink = hiWhite
		}
		a.text(dst, m.title, r.Min.X+7, r.Min.Y+2, ink)
	}

	status := "Cards Left: " + strconv.Itoa(t.g.CardsLeft())
	if t.settings.GameNumber {
		// Windows put the game number in the title bar alone. A tiled
		// window or a phone's home screen shows no title bar, so it can be
		// shown beside the count instead.
		status = "Game #" + strconv.Itoa(t.g.Number()) + "   " + status
	}
	a.text(dst, status, screenW-8-glyphW*len(status), 3, darkBlack)
}

func (a *art) drawOpenMenu(dst surface, t *table) {
	if t.menuOpen < 0 {
		return
	}
	th := themeFor(t.edition)
	menus := t.menus()
	r := menuRect(menus, t.menuOpen)
	a.fillRect(dst, r, th.dropdown)
	if t.edition == edition.WinXP {
		a.bevel(dst, r, th.disabled, th.disabled)
	} else {
		a.bevel(dst, r, th.face, darkBlack)
		a.bevel(dst, r.Inset(1), hiWhite, shadowGrey)
	}

	for j, it := range menus[t.menuOpen].items {
		ir := menuItemRect(menus, t.menuOpen, j)
		if it.separator() {
			mid := ir.Min.Y + menuSepH/2 - 1
			a.fillRect(dst, image.Rect(ir.Min.X+1, mid, ir.Max.X-1, mid+1), shadowGrey)
			a.fillRect(dst, image.Rect(ir.Min.X+1, mid+1, ir.Max.X-1, mid+2), hiWhite)
			continue
		}
		ink := darkBlack
		if j == t.menuHover {
			a.fillRect(dst, ir, th.highlight)
			ink = hiWhite
		}
		tx, ty := ir.Min.X+menuTextX-menuFrame, ir.Min.Y+2
		sx := ir.Max.X - menuTextX/2 - len(it.shortcut)*glyphW
		if !it.canRun() {
			if th.embossed {
				a.text(dst, it.label, tx+1, ty+1, hiWhite)
				a.text(dst, it.shortcut, sx+1, ty+1, hiWhite)
			}
			ink = th.disabled
		}
		a.text(dst, it.label, tx, ty, ink)
		a.text(dst, it.shortcut, sx, ty, ink)
	}
}
