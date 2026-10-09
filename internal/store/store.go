// Package store saves the settings and statistics that outlive a game, as
// one small JSON document: in a file for the desktop build, and in the
// browser's local storage for the web build.
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/thomaslaurenson/cellmate/internal/stats"
)

// version is the document format. A document from a newer cellmate is
// refused rather than half-read, so saving cannot drop fields it does not
// know about.
const version = 1

// ErrNewerVersion is returned for a document written by a newer cellmate.
var ErrNewerVersion = errors.New("saved data is from a newer version of cellmate")

// Settings are the choices made in the Options box.
type Settings struct {
	// Edition is "95" or "xp"; empty means the built-in default.
	Edition     string
	Messages    bool
	QuickPlay   bool
	DoubleClick bool
	// GameNumber shows the game number on the table as well as in the
	// title, for a window or a page that shows no title bar.
	GameNumber bool
}

// Data is everything saved between runs.
type Data struct {
	Version  int
	Settings Settings
	Stats    stats.Record
}

// Defaults returns the data for a first run: every option as Windows
// shipped it and an empty record.
func Defaults() Data {
	return Data{
		Version:  version,
		Settings: Settings{Messages: true, DoubleClick: true},
	}
}

// File keeps the data in a JSON file.
type File struct {
	path string
}

// NewFile returns a store that keeps its data at path.
func NewFile(path string) *File { return &File{path: path} }

// Load reads the file, returning the defaults if it does not exist yet.
func (f *File) Load() (Data, error) {
	b, err := os.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return Defaults(), nil
	}
	if err != nil {
		return Defaults(), fmt.Errorf("read %q: %w", f.path, err)
	}
	d, err := Decode(b)
	if err != nil {
		return d, fmt.Errorf("read %q: %w", f.path, err)
	}
	return d, nil
}

// Save writes the file, creating its directory if needed. It writes to a
// temporary file and renames it into place, so a crash mid-write leaves the
// previous save intact rather than a truncated one.
func (f *File) Save(d Data) (err error) {
	b, err := Encode(d)
	if err != nil {
		return err
	}
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %q: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".cellmate-*.json")
	if err != nil {
		return fmt.Errorf("save %q: %w", f.path, err)
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("save %q: %w", f.path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save %q: %w", f.path, err)
	}
	if err := os.Rename(tmp.Name(), f.path); err != nil {
		return fmt.Errorf("save %q: %w", f.path, err)
	}
	return nil
}
