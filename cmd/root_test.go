package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/store"
	"github.com/thomaslaurenson/cellmate/internal/ui"
)

// fakePlay records the options the root command would open the window with.
type fakePlay struct {
	called bool
	opts   ui.Options
}

func (f *fakePlay) play(_ context.Context, opts ui.Options) error {
	f.called = true
	f.opts = opts
	return nil
}

func run(t *testing.T, args ...string) (stdout, stderr string, fp *fakePlay, err error) {
	t.Helper()

	var out, errOut bytes.Buffer
	fp = &fakePlay{}
	root := NewRootCmd(&out, &errOut, fp.play, pathTo(""))
	root.SetArgs(args)
	err = root.ExecuteContext(context.Background())

	return out.String(), errOut.String(), fp, err
}

// pathTo returns a PathFunc that always names path.
func pathTo(path string) PathFunc {
	return func() (string, error) { return path, nil }
}

func TestRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		want    ui.Options
		wantErr bool
	}{
		{name: "defaults", args: nil, want: ui.Options{Edition: edition.WinXP}},
		{name: "edition flag", args: []string{"--edition", "95"}, want: ui.Options{Edition: edition.Win95}},
		{name: "game flag", args: []string{"--game", "617"}, want: ui.Options{Edition: edition.WinXP, Deal: 617}},
		{name: "game beyond 95 range", args: []string{"--edition", "95", "--game", "32001"}, wantErr: true},
		{name: "game zero", args: []string{"--game", "0"}, wantErr: true},
		{name: "impossible game", args: []string{"--game", "-2"}, want: ui.Options{Edition: edition.WinXP, Deal: -2}},
		{name: "past the impossible games", args: []string{"--game", "-3"}, wantErr: true},
		{name: "unknown edition", args: []string{"--edition", "vista"}, wantErr: true},
		{name: "stray argument", args: []string{"617"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stdout, _, fp, err := run(t, tc.args...)
			if tc.wantErr {
				if err == nil {
					t.Fatal("want an error, got nil")
				}
				if fp.called {
					t.Error("window opened despite the error")
				}
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !fp.called {
				t.Fatal("window was not opened")
			}
			if fp.opts.Edition != tc.want.Edition || fp.opts.Deal != tc.want.Deal {
				t.Errorf("options = %+v, want %+v", fp.opts, tc.want)
			}
			if !fp.opts.CanExit || fp.opts.Store != nil {
				t.Errorf("desktop options: exit %v, store %v", fp.opts.CanExit, fp.opts.Store)
			}
		})
	}
}

func TestDeal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		args       []string
		firstRow   string
		fails      bool
		rangeError bool
	}{
		{name: "default edition", args: []string{"deal", "1"}, firstRow: "JD 2D 9H JC 5D 7H 7C 5H"},
		{name: "xp only deal", args: []string{"deal", "1000000"}},
		{name: "impossible deal", args: []string{"--edition", "95", "deal", "--", "-2"}, firstRow: "AS AH AD AC 7S 7H 7D 7C"},
		{name: "edition flag limits range", args: []string{"--edition", "95", "deal", "32001"}, fails: true, rangeError: true},
		{name: "not a number", args: []string{"deal", "twelve"}, fails: true},
		{name: "trailing junk", args: []string{"deal", "12abc"}, fails: true},
		{name: "missing number", args: []string{"deal"}, fails: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stdout, _, fp, err := run(t, tc.args...)
			if fp.called {
				t.Error("deal opened the window")
			}
			if tc.fails {
				if err == nil {
					t.Fatal("want an error, got nil")
				}
				var re *edition.DealRangeError
				if tc.rangeError && !errors.As(err, &re) {
					t.Errorf("error = %v, want *edition.DealRangeError", err)
				}
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			rows := strings.Split(strings.TrimSpace(stdout), "\n")
			if len(rows) != 7 {
				t.Fatalf("got %d rows, want 7", len(rows))
			}
			if tc.firstRow != "" && rows[0] != tc.firstRow {
				t.Errorf("first row = %q, want %q", rows[0], tc.firstRow)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()
	stdout, _, _, err := run(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != Version {
		t.Errorf("version printed %q, want %q", stdout, Version)
	}
}

func TestCompletion(t *testing.T) {
	t.Parallel()
	noFiles := fmt.Sprintf(":%d\n", cobra.ShellCompDirectiveNoFileComp)
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "edition names", args: []string{"--edition", ""}, want: []string{"95", "xp"}},
		{name: "no files for a game number", args: []string{"--game", ""}},
		{name: "no files for a deal number", args: []string{"deal", ""}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stdout, _, _, err := run(t, append([]string{"__complete"}, tc.args...)...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(stdout, noFiles) {
				t.Errorf("completion output does not rule out files:\n%s", stdout)
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout, want+"\n") {
					t.Errorf("completion output missing %q:\n%s", want, stdout)
				}
			}
		})
	}
}

func runWithData(t *testing.T, path string, args ...string) (*fakePlay, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	fp := &fakePlay{}
	root := NewRootCmd(&out, &errOut, fp.play, pathTo(path))
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return fp, errOut.String(), err
}

func TestSavedData(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "cellmate.json")
	d := store.Defaults()
	d.Settings.Edition = "95"
	d.Stats.Win()
	if err := store.NewFile(path).Save(d); err != nil {
		t.Fatal(err)
	}

	fp, _, err := runWithData(t, path)
	if err != nil {
		t.Fatal(err)
	}
	if fp.opts.Edition != edition.Win95 || fp.opts.Saved.Stats.Won != 1 || fp.opts.Store == nil {
		t.Errorf("saved data not passed on: %+v", fp.opts)
	}

	fp, _, err = runWithData(t, path, "--edition", "xp")
	if err != nil {
		t.Fatal(err)
	}
	if fp.opts.Edition != edition.WinXP {
		t.Errorf("--edition did not override the saved edition: %v", fp.opts.Edition)
	}
}

func TestCorruptSavedData(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "cellmate.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp, stderr, err := runWithData(t, path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stderr, "[!] ") {
		t.Errorf("stderr = %q, want a warning", stderr)
	}
	if fp.opts.Store != nil {
		t.Error("a corrupt file would be overwritten")
	}
}

func TestNoConfigDirectory(t *testing.T) {
	t.Parallel()
	noDir := func() (string, error) { return "", errors.New("no configuration directory") }
	tests := []struct {
		name        string
		args        []string
		wantWarning bool
	}{
		{name: "playing warns and saves nothing", wantWarning: true},
		{name: "deal does not look", args: []string{"deal", "1"}},
		{name: "version does not look", args: []string{"version"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out, errOut bytes.Buffer
			fp := &fakePlay{}
			root := NewRootCmd(&out, &errOut, fp.play, noDir)
			root.SetArgs(tc.args)
			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			if got := strings.HasPrefix(errOut.String(), "[!] "); got != tc.wantWarning {
				t.Errorf("stderr = %q, want a warning: %v", errOut.String(), tc.wantWarning)
			}
			if fp.called && fp.opts.Store != nil {
				t.Error("the game was given a store with nowhere to save")
			}
		})
	}
}

func TestResolveEdition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		flag    string
		flagSet bool
		saved   string
		want    edition.Edition
	}{
		{name: "nothing saved", flag: "xp", want: edition.WinXP},
		{name: "saved wins over default", flag: "xp", saved: "95", want: edition.Win95},
		{name: "flag wins over saved", flag: "xp", flagSet: true, saved: "95", want: edition.WinXP},
		{name: "unknown saved value ignored", flag: "xp", saved: "vista", want: edition.Default},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveEdition(tc.flag, tc.flagSet, tc.saved)
			if err != nil || got != tc.want {
				t.Errorf("resolveEdition = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}

func TestDataPath(t *testing.T) {
	t.Parallel()
	p, err := DataPath()
	if err != nil {
		t.Skipf("no configuration directory here: %v", err)
	}
	if filepath.Base(p) != "cellmate.json" || filepath.Base(filepath.Dir(p)) != "cellmate" {
		t.Errorf("DataPath() = %q", p)
	}
}

// TestDealNegativeNumberNeedsTheDashes covers the trap in asking for an
// impossible deal: "-1" parses as a flag, so the error must say what to
// type instead, and only when what was typed is a number.
func TestDealNegativeNumberNeedsTheDashes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		arg  string
		hint string
	}{
		{name: "impossible deal", arg: "-1", hint: "deal -- -1"},
		{name: "other negative number", arg: "-12", hint: "deal -- -12"},
		{name: "mistyped flag", arg: "-x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stdout, _, _, err := run(t, "deal", tc.arg)
			var ne *pflag.NotExistError
			if !errors.As(err, &ne) {
				t.Fatalf("error = %v, want *pflag.NotExistError", err)
			}
			hinted := strings.Contains(err.Error(), "goes after --")
			if tc.hint == "" && hinted {
				t.Errorf("error = %v, want no negative deal hint", err)
			}
			if tc.hint != "" && !strings.Contains(err.Error(), tc.hint) {
				t.Errorf("error = %v, want the hint to write %s", err, tc.hint)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
		})
	}
}
