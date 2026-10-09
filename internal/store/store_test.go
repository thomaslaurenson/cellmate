package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    Settings
		won     int
		wantErr error
		bad     bool
	}{
		{name: "missing options keep defaults", input: `{"version":1,"stats":{"won":3}}`,
			want: Settings{Messages: true, DoubleClick: true}, won: 3},
		{name: "options switched off stay off", input: `{"version":1,"settings":{"messages":false,"doubleClick":false,"quickPlay":true,"edition":"95"}}`,
			want: Settings{Edition: "95", QuickPlay: true}},
		{name: "the game number is off unless asked for", input: `{"version":1,"settings":{"gameNumber":true}}`,
			want: Settings{Messages: true, DoubleClick: true, GameNumber: true}},
		{name: "newer version refused", input: `{"version":2}`, wantErr: ErrNewerVersion,
			want: Settings{Messages: true, DoubleClick: true}},
		{name: "not json", input: `{`, bad: true, want: Settings{Messages: true, DoubleClick: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d, err := Decode([]byte(tc.input))
			switch {
			case tc.wantErr != nil:
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %v, want %v", err, tc.wantErr)
				}
			case tc.bad:
				if err == nil {
					t.Fatal("want an error for malformed input")
				}
			case err != nil:
				t.Fatalf("unexpected error: %v", err)
			}
			if d.Settings != tc.want || d.Stats.Won != tc.won {
				t.Errorf("decoded %+v, want settings %+v and %d won", d, tc.want, tc.won)
			}
		})
	}
}

func TestFileRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "cellmate.json")
	f := NewFile(path)

	d, err := f.Load()
	if err != nil {
		t.Fatalf("loading a missing file: %v", err)
	}
	if d != Defaults() {
		t.Errorf("missing file loaded %+v, want defaults", d)
	}

	d.Stats.Win()
	d.Settings.QuickPlay = true
	if err := f.Save(d); err != nil {
		t.Fatal(err)
	}
	got, err := f.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != d {
		t.Errorf("loaded %+v, want %+v", got, d)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("save left %d files behind, want just the one", len(entries))
	}
}

func TestFileLoadCorrupt(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "cellmate.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := NewFile(path).Load()
	if err == nil {
		t.Fatal("want an error for a corrupt file")
	}
	if d != Defaults() {
		t.Error("a corrupt file should still hand back defaults to play with")
	}
}

func TestFileLoadUnreadable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, err := NewFile(dir).Load(); err == nil {
		t.Error("reading a directory as the data file succeeded")
	}
}

func TestFileSaveFailure(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := NewFile(filepath.Join(blocker, "cellmate.json")).Save(Defaults()); err == nil {
		t.Error("saving under a regular file succeeded")
	}
}

func TestFileSaveOntoDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "cellmate.json")
	if err := os.MkdirAll(filepath.Join(path, "occupied"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := NewFile(path).Save(Defaults()); err == nil {
		t.Fatal("saving over a non-empty directory succeeded")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("a failed save left its temporary file behind: %d entries", len(entries))
	}
}
