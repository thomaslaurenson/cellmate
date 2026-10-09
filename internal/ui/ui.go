// Package ui draws the FreeCell table in the style of the Windows versions
// and turns mouse and keyboard input into moves.
package ui

import (
	"context"
	"fmt"
	"image"
	"math"
	"math/rand/v2"
	"sync/atomic"

	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/game"
	"github.com/thomaslaurenson/cellmate/internal/gfx"
	"github.com/thomaslaurenson/cellmate/internal/stats"
	"github.com/thomaslaurenson/cellmate/internal/store"
)

// Saver keeps settings and statistics between runs.
type Saver interface {
	Save(store.Data) error
}

// Options chooses how the game starts.
type Options struct {
	Edition edition.Edition
	// Deal is the deal to open with; zero picks one at random.
	Deal int
	// Saved is what the last run left behind; its edition is ignored in
	// favour of Edition, which the caller has already resolved.
	Saved store.Data
	// Store receives the data whenever it changes; nil keeps nothing.
	Store Saver
	// CanExit offers Exit in the Game menu. A browser tab is closed by the
	// browser, so the web build leaves it out.
	CanExit bool
	// Version is shown in the About box.
	Version string
}

// Run opens the game window and blocks until the window is closed or ctx is
// cancelled, returning ctx.Err() in the second case so the caller can tell an
// interrupt from a normal exit.
func Run(ctx context.Context, opts Options) error {
	a, err := newArt()
	if err != nil {
		return err
	}
	w := &window{t: newTable(opts, gfx.SetTitle), art: a}
	stop := context.AfterFunc(ctx, w.cancel)
	defer stop()
	cfg := gfx.Config{Width: screenW, Height: screenH}
	if err := gfx.Run(cfg, w); err != nil {
		return err
	}
	return ctx.Err()
}

// window adapts the table to the frame loop, turning each tick's raw input
// into the value the table steps on.
type window struct {
	t   *table
	art *art
	// cancelled is set when the caller's context ends. The frame loop calls
	// Update with no context of its own, so the cancellation reaches the
	// next tick as this flag.
	cancelled atomic.Bool
}

var _ gfx.Game = (*window)(nil)

// cancel asks the frame loop to stop at its next tick. It is safe to call
// from any goroutine.
func (w *window) cancel() { w.cancelled.Store(true) }

// Update advances the table by one tick.
func (w *window) Update(in gfx.Input) error {
	if w.cancelled.Load() || w.t.quit {
		return gfx.ErrQuit
	}
	w.t.step(readInput(in))
	return nil
}

// Draw paints the table.
func (w *window) Draw(c *gfx.Canvas) { w.art.drawTable(c, w.t) }

// input is one tick's worth of mouse and keyboard state. Collecting it in a
// value keeps the table testable without a window.
type input struct {
	cursor       image.Point
	leftPressed  bool
	leftReleased bool
	rightDown    bool
	enter        bool
	escape       bool
	backspace    bool
	chars        []rune
	ctrl, shift  bool
	// touch marks input from a finger, which hits less precisely than a
	// mouse and so is given a wider target.
	touch bool
	// key is the function key pressed this tick, one of functionKeys, or
	// zero for none.
	key gfx.Key
	// close is the player asking the window to close.
	close bool
}

// functionKeys are the menu shortcuts, checked in order each tick.
var functionKeys = []gfx.Key{
	gfx.KeyF1, gfx.KeyF2, gfx.KeyF3, gfx.KeyF4, gfx.KeyF5, gfx.KeyF10,
}

// readInput narrows a tick of platform input to what the table reads.
func readInput(raw gfx.Input) input {
	in := input{
		cursor:       raw.Cursor,
		leftPressed:  raw.ButtonPressed(gfx.ButtonLeft),
		leftReleased: raw.ButtonReleased(gfx.ButtonLeft),
		rightDown:    raw.ButtonDown(gfx.ButtonRight),
		enter:        raw.KeyPressed(gfx.KeyEnter) || raw.KeyPressed(gfx.KeyNumpadEnter),
		escape:       raw.KeyPressed(gfx.KeyEscape),
		backspace:    repeating(raw, gfx.KeyBackspace),
		chars:        append([]rune(nil), raw.Chars...),
		ctrl:         raw.KeyDown(gfx.KeyControl),
		shift:        raw.KeyDown(gfx.KeyShift),
		touch:        raw.Touch,
		close:        raw.CloseRequested,
	}
	for _, k := range functionKeys {
		if raw.KeyPressed(k) {
			in.key = k
		}
	}
	return in
}

// repeating reports a key press, repeating while it is held the way a text
// box does: once at once, then steadily after a short delay.
func repeating(raw gfx.Input, k gfx.Key) bool {
	d := raw.KeyTicks(k)
	return d == 1 || (d > 30 && d%3 == 0)
}

// Timing, in ticks of the frame loop, which runs at 60 a second.
const (
	// doubleClickTicks matches the Windows default double-click time of
	// 500 milliseconds.
	doubleClickTicks = 30
	// flightTicks is how long a card takes to fly home.
	flightTicks = 8
	// winTicks is how long the king celebrates a win before the message.
	winTicks = 100
)

// winKing returns where the celebrating king is drawn, and how large, at
// tick n of the win: he grows from his place between the cells to fill the
// middle of the table, bouncing as he goes.
func winKing(n int) (image.Point, float64) {
	p := min(float64(n)/winTicks, 1)
	ease := 1 - (1-p)*(1-p)
	start := kingRect().Min.Add(kingRect().Size().Div(2))
	end := image.Pt(screenW/2, (colTop+screenH)/2)
	bounce := math.Abs(math.Sin(p*3*math.Pi)) * 24 * (1 - p)
	x := float64(start.X) + float64(end.X-start.X)*ease
	y := float64(start.Y) + float64(end.Y-start.Y)*ease - bounce
	return image.Pt(int(x), int(y)), 1 + 5*ease
}

// flight is a card on its way home, drawn between its old and new places.
type flight struct {
	move     game.Move
	from, to image.Point
	tick     int
}

func (f *flight) at() image.Point {
	t := float64(f.tick) / flightTicks
	return image.Pt(
		f.from.X+int(float64(f.to.X-f.from.X)*t),
		f.from.Y+int(float64(f.to.Y-f.from.Y)*t),
	)
}

// reveal is a buried card shown in full while the right button is held.
type reveal struct {
	column, index int
}

// table is the game in progress and everything the player is doing to it.
type table struct {
	edition  edition.Edition
	g        *game.Game
	setTitle func(string)
	randIntN func(int) int

	settings store.Settings
	total    stats.Record
	session  stats.Record
	store    Saver
	canExit  bool
	version  string
	// saveFailed stops a broken store raising a message after every game.
	saveFailed bool
	quit       bool

	menuOpen  int
	menuHover int

	tick       int
	cursor     image.Point
	selected   *game.Pos
	selectTick int
	// winTick counts through the win animation; zero when none is playing.
	winTick int
	flight  *flight
	reveal  *reveal
	dialog  *dialog
	// changed marks a board that has not been checked for a win or a dead
	// end since it last changed; over stops the same ending being reported
	// twice.
	changed bool
	over    bool
}

func newTable(opts Options, setTitle func(string)) *table {
	t := &table{
		edition:   opts.Edition,
		setTitle:  setTitle,
		randIntN:  rand.IntN,
		settings:  opts.Saved.Settings,
		total:     opts.Saved.Stats,
		store:     opts.Store,
		canExit:   opts.CanExit,
		version:   opts.Version,
		menuOpen:  -1,
		menuHover: -1,
	}
	n := opts.Deal
	if n == 0 {
		n = t.randomDeal()
	}
	t.newDeal(n)
	return t
}

func (t *table) randomDeal() int {
	return t.randIntN(t.edition.MaxDeal()) + 1
}

func (t *table) newDeal(n int) {
	t.g = game.New(n)
	t.selected, t.flight, t.reveal = nil, nil, nil
	t.over, t.winTick = false, 0
	t.setTitle(fmt.Sprintf("cellmate Game #%d", n))
	t.autoplay()
}
