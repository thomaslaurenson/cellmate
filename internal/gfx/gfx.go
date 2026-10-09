package gfx

import (
	"errors"
	"image"
)

// ErrQuit is returned by Game.Update to stop the frame loop. Run treats it as
// an ordinary finish and returns nil.
var ErrQuit = errors.New("quit")

// Key identifies a keyboard key. The zero value names no key, so a field
// holding "the key pressed this tick" reads as empty until one is.
type Key int

// The keys the game reads. Only these are reported; anything else the player
// presses reaches the game as a character in Input.Chars, or not at all.
const (
	KeyNone Key = iota
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF10
	KeyEnter
	KeyNumpadEnter
	KeyEscape
	KeyBackspace
	KeyControl
	KeyShift
)

// Button identifies a mouse button.
type Button int

// The mouse buttons the game reads.
const (
	ButtonLeft Button = iota
	ButtonRight
)

// Input is one tick of mouse and keyboard state, passed to Game.Update.
//
// Presses and releases are counted by the platform rather than sampled, so a
// click or keystroke that begins and ends between two ticks is still
// reported. The zero value is a tick in which nothing happened.
type Input struct {
	// Cursor is the pointer in logical screen coordinates.
	Cursor image.Point
	// Held reports the buttons currently down.
	Held map[Button]bool
	// Pressed and Released report buttons that changed during this tick.
	Pressed  map[Button]bool
	Released map[Button]bool
	// Keys holds how many ticks each key has been held for, so 1 means the
	// key went down during this tick.
	Keys map[Key]int
	// Chars are the characters typed during this tick.
	Chars []rune
	// Touch reports that this tick's buttons came from a finger rather than
	// a mouse. A finger is far less precise, so the game widens what counts
	// as a hit rather than redrawing anything.
	Touch bool
	// CloseRequested reports that the player asked the window to close
	// during this tick. The window stays open until Update returns ErrQuit,
	// so the game can ask first. A browser tab never reports it.
	CloseRequested bool
}

// KeyDown reports whether k is held.
func (in Input) KeyDown(k Key) bool { return in.Keys[k] > 0 }

// KeyPressed reports whether k went down during this tick.
func (in Input) KeyPressed(k Key) bool { return in.Keys[k] == 1 }

// KeyTicks reports how many ticks k has been held for, which a caller needs
// to repeat a key the way a text box does.
func (in Input) KeyTicks(k Key) int { return in.Keys[k] }

// ButtonDown reports whether b is held.
func (in Input) ButtonDown(b Button) bool { return in.Held[b] }

// ButtonPressed reports whether b went down during this tick.
func (in Input) ButtonPressed(b Button) bool { return in.Pressed[b] }

// ButtonReleased reports whether b came up during this tick.
func (in Input) ButtonReleased(b Button) bool { return in.Released[b] }

// Config describes the window or canvas Run opens.
type Config struct {
	// Width and Height are the logical screen size. The platform scales it
	// to whatever the player's window or page gives it.
	Width, Height int
	// Title is the window title, or the page title in a browser.
	Title string
}

// Game is the loop Run drives: one Update per tick at 60 ticks a second,
// then one Draw per displayed frame.
type Game interface {
	Update(in Input) error
	Draw(c *Canvas)
}
