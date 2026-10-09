//go:build !(js && wasm)

package gfx

import (
	"errors"
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// SetTitle sets the window title.
func SetTitle(title string) { ebiten.SetWindowTitle(title) }

// keyCodes maps the keys the game reads onto Ebitengine's. A key with two
// physical positions appears once for each.
var keyCodes = map[Key][]ebiten.Key{
	KeyF1:          {ebiten.KeyF1},
	KeyF2:          {ebiten.KeyF2},
	KeyF3:          {ebiten.KeyF3},
	KeyF4:          {ebiten.KeyF4},
	KeyF5:          {ebiten.KeyF5},
	KeyF10:         {ebiten.KeyF10},
	KeyEnter:       {ebiten.KeyEnter},
	KeyNumpadEnter: {ebiten.KeyNumpadEnter},
	KeyEscape:      {ebiten.KeyEscape},
	KeyBackspace:   {ebiten.KeyBackspace},
	KeyControl:     {ebiten.KeyControlLeft, ebiten.KeyControlRight},
	KeyShift:       {ebiten.KeyShiftLeft, ebiten.KeyShiftRight},
}

var buttonCodes = map[Button]ebiten.MouseButton{
	ButtonLeft:  ebiten.MouseButtonLeft,
	ButtonRight: ebiten.MouseButtonRight,
}

// window adapts a Game to Ebitengine's own loop. It holds the maps the Input
// value is rebuilt into each tick, so a tick allocates nothing.
type window struct {
	game   Game
	cfg    Config
	canvas *Canvas
	in     Input
	// scaleFactor reports the monitor's device scale factor: how many of
	// its pixels make one of the device-independent pixels Ebitengine
	// measures the window in. Tests supply their own.
	scaleFactor func() float64
}

var (
	_ ebiten.Game      = (*window)(nil)
	_ ebiten.LayoutFer = (*window)(nil)
)

// Run opens a window, drives g's loop against it, and blocks until g stops
// or fails.
func Run(cfg Config, g Game) error {
	w := &window{
		game:   g,
		cfg:    cfg,
		canvas: newCanvas(image.Pt(cfg.Width, cfg.Height)),
		in: Input{
			Held:     make(map[Button]bool),
			Pressed:  make(map[Button]bool),
			Released: make(map[Button]bool),
			Keys:     make(map[Key]int),
		},
		scaleFactor: monitorScaleFactor,
	}
	ebiten.SetWindowSize(cfg.Width, cfg.Height)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	if cfg.Title != "" {
		SetTitle(cfg.Title)
	}
	if err := ebiten.RunGame(w); err != nil {
		return fmt.Errorf("run game window: %w", err)
	}
	return nil
}

// monitorScaleFactor is the current monitor's device scale factor, or 1
// when there is no monitor to ask.
func monitorScaleFactor() float64 {
	if m := ebiten.Monitor(); m != nil {
		return m.DeviceScaleFactor()
	}
	return 1
}

// Update reads this tick's input and hands it to the game.
func (w *window) Update() error {
	x, y := ebiten.CursorPosition()
	w.in.Cursor = w.canvas.view.logical(float64(x), float64(y))

	for k, codes := range keyCodes {
		ticks := 0
		for _, code := range codes {
			ticks = max(ticks, inpututil.KeyPressDuration(code))
		}
		w.in.Keys[k] = ticks
	}
	for b, code := range buttonCodes {
		w.in.Held[b] = ebiten.IsMouseButtonPressed(code)
		w.in.Pressed[b] = inpututil.IsMouseButtonJustPressed(code)
		w.in.Released[b] = inpututil.IsMouseButtonJustReleased(code)
	}
	w.in.Chars = ebiten.AppendInputChars(w.in.Chars[:0])
	w.in.CloseRequested = ebiten.IsWindowBeingClosed()

	if err := w.game.Update(w.in); err != nil {
		if errors.Is(err, ErrQuit) {
			return ebiten.Termination
		}
		return err
	}
	return nil
}

// Draw paints one frame.
func (w *window) Draw(screen *ebiten.Image) {
	w.canvas.begin(screen)
	w.game.Draw(w.canvas)
}

// LayoutF gives the game the whole window, at the monitor's own resolution.
// Ebitengine would otherwise scale a fixed logical screen to the window
// itself, and blur it whenever the two differ by a fraction; drawing at full
// size is what keeps the board sharp at any window size. The canvas fits the
// board into whatever this returns.
func (w *window) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	s := w.scaleFactor()
	if s <= 0 {
		s = 1
	}
	return max(outsideWidth*s, 1), max(outsideHeight*s, 1)
}

// Layout is the whole-pixel form the Game interface asks for. Ebitengine
// calls LayoutF instead when a game has one; this rounds the same answer up
// the way Ebitengine does.
func (w *window) Layout(outsideWidth, outsideHeight int) (int, int) {
	fw, fh := w.LayoutF(float64(outsideWidth), float64(outsideHeight))
	return int(math.Ceil(fw)), int(math.Ceil(fh))
}
