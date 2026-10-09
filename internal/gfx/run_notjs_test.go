//go:build !(js && wasm)

package gfx

import (
	"errors"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// stubGame records the input it was given and returns whatever it is told to.
type stubGame struct {
	got  Input
	err  error
	runs int
}

func (g *stubGame) Update(in Input) error {
	g.got = in
	g.runs++
	return g.err
}

func (g *stubGame) Draw(*Canvas) {}

// newTestWindow builds a window with the maps Run would have made for it.
func newTestWindow(g Game) *window {
	return &window{
		game:   g,
		cfg:    Config{Width: 640, Height: 480},
		canvas: newCanvas(image.Pt(640, 480)),
		in: Input{
			Held:     make(map[Button]bool),
			Pressed:  make(map[Button]bool),
			Released: make(map[Button]bool),
			Keys:     make(map[Key]int),
		},
		scaleFactor: func() float64 { return 1 },
	}
}

// TestUpdateMapsTheCursorOntoTheBoard checks that the cursor reaches the
// game in board coordinates rather than the window's. Outside a running
// game Ebitengine reports the cursor at the window's top left corner, which
// on a screen with a margin beside the board is a point off its left edge.
func TestUpdateMapsTheCursorOntoTheBoard(t *testing.T) {
	t.Parallel()
	g := &stubGame{}
	w := newTestWindow(g)
	// Twice the board's size, with a margin of 100 either side of it
	w.canvas.begin(ebiten.NewImage(1480, 960))

	if err := w.Update(); err != nil {
		t.Fatalf("Update returned %v", err)
	}
	if want := image.Pt(-50, 0); g.got.Cursor != want {
		t.Errorf("cursor = %v, want %v", g.got.Cursor, want)
	}
}

// TestUpdateReportsEveryKeyTheGameCanRead guards against a key being added to
// the Key set and not to this platform's map. A missing key is silent: it
// simply never fires, and only the platform that forgot it is affected.
func TestUpdateReportsEveryKeyTheGameCanRead(t *testing.T) {
	t.Parallel()
	g := &stubGame{}
	w := newTestWindow(g)

	if err := w.Update(); err != nil {
		t.Fatalf("Update returned %v", err)
	}
	for k := KeyNone + 1; k <= KeyShift; k++ {
		if _, ok := g.got.Keys[k]; !ok {
			t.Errorf("key %d is missing from the desktop key map", k)
		}
	}
	if _, ok := g.got.Keys[KeyNone]; ok {
		t.Error("KeyNone was reported as a real key")
	}
}

func TestUpdateReportsBothMouseButtons(t *testing.T) {
	t.Parallel()
	g := &stubGame{}
	w := newTestWindow(g)

	if err := w.Update(); err != nil {
		t.Fatalf("Update returned %v", err)
	}
	for _, b := range []Button{ButtonLeft, ButtonRight} {
		if _, ok := g.got.Held[b]; !ok {
			t.Errorf("button %d is missing from the held map", b)
		}
		if _, ok := g.got.Pressed[b]; !ok {
			t.Errorf("button %d is missing from the pressed map", b)
		}
		if _, ok := g.got.Released[b]; !ok {
			t.Errorf("button %d is missing from the released map", b)
		}
	}
}

func TestUpdateTranslatesTheGamesResult(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	tests := []struct {
		name string
		give error
		want error
	}{
		{name: "carrying on", give: nil, want: nil},
		{name: "quitting", give: ErrQuit, want: ebiten.Termination},
		{name: "failing", give: boom, want: boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &stubGame{err: tc.give}
			w := newTestWindow(g)

			err := w.Update()
			if !errors.Is(err, tc.want) {
				t.Fatalf("Update returned %v, want %v", err, tc.want)
			}
			if g.runs != 1 {
				t.Errorf("the game was updated %d times, want once", g.runs)
			}
		})
	}
}
