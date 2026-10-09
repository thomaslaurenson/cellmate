package ui

import (
	"errors"
	"image"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/gfx"
	"github.com/thomaslaurenson/cellmate/internal/store"
)

// newTestWindow builds a window around a table on a known deal.
func newTestWindow(t *testing.T) *window {
	t.Helper()
	tb := newTable(Options{Edition: edition.WinXP, Deal: 1, Saved: store.Defaults()}, func(string) {})
	tb.randIntN = func(int) int { return 616 }
	return &window{t: tb, art: newTestArt(t)}
}

func TestWindowUpdateStopsWhenAsked(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		prepare  func(*window)
		wantQuit bool
	}{
		{
			name:    "runs on while nothing has happened",
			prepare: func(*window) {},
		},
		{
			name:     "stops when the context is cancelled",
			prepare:  func(w *window) { w.cancel() },
			wantQuit: true,
		},
		{
			name:     "stops when the game asks to quit",
			prepare:  func(w *window) { w.t.quit = true },
			wantQuit: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newTestWindow(t)
			tc.prepare(w)

			err := w.Update(gfx.Input{})
			if got := errors.Is(err, gfx.ErrQuit); got != tc.wantQuit {
				t.Fatalf("Update returned %v, want ErrQuit = %v", err, tc.wantQuit)
			}
			if !tc.wantQuit && err != nil {
				t.Errorf("Update returned %v, want nil", err)
			}
		})
	}
}

func TestWindowUpdateStepsTheTable(t *testing.T) {
	t.Parallel()
	w := newTestWindow(t)

	// A cursor position must reach the table, which proves Update translated
	// the platform input rather than stepping the table with a zero value.
	if err := w.Update(gfx.Input{Cursor: image.Pt(321, 123)}); err != nil {
		t.Fatalf("Update returned %v", err)
	}
	if w.t.cursor != image.Pt(321, 123) {
		t.Errorf("table cursor = %v, want the position Update was given", w.t.cursor)
	}

	// F5 opens the Options box, so a function key must survive the trip too
	if err := w.Update(gfx.Input{Keys: map[gfx.Key]int{gfx.KeyF5: 1}}); err != nil {
		t.Fatalf("Update returned %v", err)
	}
	if w.t.dialog == nil {
		t.Fatal("F5 did not open a dialog")
	}
	if got := w.t.dialog.title; got != "Options" {
		t.Errorf("F5 opened %q, want the Options box", got)
	}
}
