//go:build js && wasm

package gfx

import (
	"errors"
	"image"
	"math"
	"syscall/js"
)

// SetTitle sets the page title.
func SetTitle(title string) { js.Global().Get("document").Set("title", title) }

// driver holds one run's worth of canvas and input state. It exists so that
// state belongs to a running game rather than to the package, which lets a
// second run start clean and keeps anything else from reading it.
type driver struct {
	cfg    Config
	canvas js.Value
	screen *Canvas

	cursor image.Point

	// keyHeld is what the browser reports as down right now; keyTicks is how
	// many ticks the game has seen the key down for. keyPresses counts
	// presses so a keystroke that begins and ends between two ticks is not
	// lost, which is the same reason the buttons are counted rather than
	// sampled.
	keyHeld    map[Key]bool
	keyTicks   map[Key]int
	keyPresses map[Key]int

	held      map[Button]bool
	presses   map[Button]int
	releases  map[Button]int
	tickInput Input

	// finger carries a touch screen's single contact, which reaches the
	// game as the left and right buttons it has no way to send directly.
	finger touch

	pending []rune
	typed   []rune
}

var keyByCode = map[string]Key{
	"F1":          KeyF1,
	"F2":          KeyF2,
	"F3":          KeyF3,
	"F4":          KeyF4,
	"F5":          KeyF5,
	"F10":         KeyF10,
	"Enter":       KeyEnter,
	"NumpadEnter": KeyNumpadEnter,
	"Escape":      KeyEscape,
	"Backspace":   KeyBackspace,
	"ControlLeft": KeyControl, "ControlRight": KeyControl,
	"ShiftLeft": KeyShift, "ShiftRight": KeyShift,
}

// Run opens a canvas on the page, drives g's loop against it, and blocks
// until g stops or fails.
func Run(cfg Config, g Game) error {
	d := &driver{
		cfg:        cfg,
		keyHeld:    make(map[Key]bool),
		keyTicks:   make(map[Key]int),
		keyPresses: make(map[Key]int),
		held:       make(map[Button]bool),
		presses:    make(map[Button]int),
		releases:   make(map[Button]int),
	}
	// The reported maps are separate from the ones the events write, because
	// a tick combines a mouse and a finger rather than reporting either one.
	d.tickInput = Input{
		Held:     make(map[Button]bool),
		Pressed:  make(map[Button]bool),
		Released: make(map[Button]bool),
		Keys:     d.keyTicks,
	}
	for _, b := range []Button{ButtonLeft, ButtonRight} {
		d.held[b], d.presses[b], d.releases[b] = false, 0, 0
	}
	if cfg.Title != "" {
		SetTitle(cfg.Title)
	}
	d.open()
	d.listen()
	return d.loop(g)
}

// open creates the canvas and adds it to the page.
func (d *driver) open() {
	doc := js.Global().Get("document")
	d.canvas = doc.Call("createElement", "canvas")
	// The canvas covers the page, and fitCanvas gives its bitmap as many
	// pixels as it covers. Keys are read from the window, so the canvas
	// never takes focus, and without focus the browser draws no outline
	// over the board's edge.
	d.canvas.Get("style").Set("cssText",
		"display:block;width:100%;height:100%;outline:none;touch-action:none")
	doc.Get("body").Call("appendChild", d.canvas)

	// An opaque context lets the browser skip compositing the page behind
	// the board, which is always painted every frame anyway.
	ctx := d.canvas.Call("getContext", "2d", map[string]any{"alpha": false})
	d.screen = newCanvas(ctx, image.Pt(d.cfg.Width, d.cfg.Height))
	d.fitCanvas()
}

// fitCanvas gives the canvas's bitmap one pixel for every device pixel it
// covers, which changes when the window is resized, a phone is turned or the
// page is zoomed. A bitmap smaller than its box is stretched by the browser,
// and that stretching is the blur the board is drawn at full size to avoid.
func (d *driver) fitCanvas() {
	dpr := js.Global().Get("devicePixelRatio").Float()
	if dpr <= 0 {
		dpr = 1
	}
	w := max(int(math.Round(d.canvas.Get("clientWidth").Float()*dpr)), 1)
	h := max(int(math.Round(d.canvas.Get("clientHeight").Float()*dpr)), 1)
	if d.canvas.Get("width").Int() != w || d.canvas.Get("height").Int() != h {
		d.canvas.Set("width", w)
		d.canvas.Set("height", h)
	}
	d.screen.begin(w, h)
}

// on registers fn as a listener for event on target.
func on(target js.Value, event string, fn func(js.Value)) {
	target.Call("addEventListener", event, js.FuncOf(func(_ js.Value, args []js.Value) any {
		fn(args[0])
		return nil
	}))
}

// buttonOf maps a mouse event's button number, ignoring the ones the game
// does not use.
func buttonOf(e js.Value) (Button, bool) {
	switch e.Get("button").Int() {
	case 0:
		return ButtonLeft, true
	case 2:
		return ButtonRight, true
	}
	return 0, false
}

// isFinger reports whether e came from a touch screen rather than a mouse or
// a pen, both of which have a hover position and a second button.
func isFinger(e js.Value) bool { return e.Get("pointerType").String() == "touch" }

// pointAt converts an event's page position into board coordinates. It fails
// for a press on the margin beside the board, and while the canvas has no
// size at all, which is the case before the page has laid it out.
func (d *driver) pointAt(e js.Value) (image.Point, bool) {
	r := d.canvas.Call("getBoundingClientRect")
	return boardPoint(
		e.Get("clientX").Float()-r.Get("left").Float(),
		e.Get("clientY").Float()-r.Get("top").Float(),
		r.Get("width").Float(), r.Get("height").Float(),
		image.Pt(d.cfg.Width, d.cfg.Height),
	)
}

// listen wires the page events the game reads.
func (d *driver) listen() {
	win := js.Global()

	// Pointer events cover a mouse, a finger and a pen in one set of
	// handlers, so the board needs no separate touch path. A finger is told
	// apart by pointerType, because it has no hover and no second button and
	// so has to reach the game through the touch state machine instead.
	on(d.canvas, "pointermove", func(e js.Value) {
		pt, ok := d.pointAt(e)
		if !ok {
			return
		}
		if isFinger(e) {
			d.finger.move(pt)
			return
		}
		d.cursor = pt
	})
	on(d.canvas, "pointerdown", func(e js.Value) {
		pt, ok := d.pointAt(e)
		if !ok {
			return
		}
		if isFinger(e) {
			// A second finger is ignored: the board has no two-finger
			// gesture, and treating one as a fresh press would cancel the
			// hold the first finger was part way through.
			if !d.finger.down {
				d.finger.begin(pt)
			}
			return
		}
		d.cursor = pt
		if b, ok := buttonOf(e); ok {
			d.held[b] = true
			d.presses[b]++
		}
	})
	// Releases are taken from the window so that letting go outside the
	// board still ends the press.
	on(win, "pointerup", func(e js.Value) {
		if isFinger(e) {
			d.finger.end()
			return
		}
		if b, ok := buttonOf(e); ok {
			d.held[b] = false
			d.releases[b]++
		}
	})
	on(win, "pointercancel", func(e js.Value) {
		if isFinger(e) {
			d.finger.cancel()
		}
	})
	// A long press on a phone would otherwise raise the browser's own
	// selection menu over the board, and a right click its context menu.
	on(d.canvas, "contextmenu", func(e js.Value) { e.Call("preventDefault") })

	// Keys come from the window rather than the canvas, which would only
	// receive them while it held focus.
	on(win, "keydown", func(e js.Value) {
		if k, ok := keyByCode[e.Get("code").String()]; ok {
			if !d.keyHeld[k] {
				d.keyPresses[k]++
			}
			d.keyHeld[k] = true
			// F5 reloads the page and F10 opens the browser's menu bar.
			// FreeCell binds both, so claim the keys the game uses and
			// leave every other key to the browser.
			e.Call("preventDefault")
		}
		if s := e.Get("key").String(); !e.Get("ctrlKey").Bool() && !e.Get("metaKey").Bool() {
			if rs := []rune(s); len(rs) == 1 {
				d.pending = append(d.pending, rs...)
			}
		}
	})
	on(win, "keyup", func(e js.Value) {
		if k, ok := keyByCode[e.Get("code").String()]; ok {
			d.keyHeld[k] = false
		}
	})
	// A key or button held as the tab loses focus would otherwise stay held
	// for ever, since its release lands on another window.
	on(win, "blur", func(js.Value) {
		for k := range d.keyHeld {
			d.keyHeld[k] = false
		}
		for b := range d.held {
			d.held[b] = false
		}
		d.finger.cancel()
	})
}

// tick advances the input state by one tick and returns it.
func (d *driver) tick() Input {
	for k := range d.keyHeld {
		switch {
		case d.keyHeld[k]:
			d.keyTicks[k]++
		case d.keyPresses[k] > 0:
			// Pressed and released inside one tick: report it as held for
			// exactly this tick, then let it go.
			d.keyTicks[k] = 1
		default:
			d.keyTicks[k] = 0
		}
		d.keyPresses[k] = 0
	}
	d.typed, d.pending = d.pending, d.typed[:0]

	// The mouse and the finger are combined fresh each tick rather than
	// latched, so a device with both keeps working either way and neither
	// can leave a button stuck down.
	ft := d.finger.tick()
	if ft.moveCursor {
		d.cursor = ft.cursor
	}
	d.tickInput.Touch = ft != (touchTick{})
	for _, b := range []Button{ButtonLeft, ButtonRight} {
		held, pressed, released := d.held[b], d.presses[b] > 0, d.releases[b] > 0
		if b == ButtonLeft {
			held = held || ft.leftPressed
			pressed = pressed || ft.leftPressed
			released = released || ft.leftReleased
		} else {
			held = held || ft.rightDown
		}
		d.tickInput.Held[b] = held
		d.tickInput.Pressed[b] = pressed
		d.tickInput.Released[b] = released
		d.presses[b], d.releases[b] = 0, 0
	}

	d.tickInput.Cursor = d.cursor
	d.tickInput.Chars = d.typed
	return d.tickInput
}

// loop drives g from requestAnimationFrame, running Update at a fixed rate
// and drawing once per displayed frame.
func (d *driver) loop(g Game) error {
	const step = 1000.0 / 60.0
	// A tab in the background stops being given frames, so the first frame
	// after it returns carries a large gap. Capping it keeps the game from
	// running hundreds of ticks at once to catch up.
	const maxGap = 250.0

	done := make(chan error, 1)
	var last, backlog float64
	var frame js.Func

	frame = js.FuncOf(func(_ js.Value, args []js.Value) any {
		now := args[0].Float()
		if last == 0 {
			last = now
		}
		backlog = min(backlog+now-last, maxGap)
		last = now

		for backlog >= step {
			backlog -= step
			if err := g.Update(d.tick()); err != nil {
				frame.Release()
				if errors.Is(err, ErrQuit) {
					done <- nil
					return nil
				}
				done <- err
				return nil
			}
		}
		d.fitCanvas()
		g.Draw(d.screen)
		js.Global().Call("requestAnimationFrame", frame)
		return nil
	})
	js.Global().Call("requestAnimationFrame", frame)
	return <-done
}
