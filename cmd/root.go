// Package cmd wires the cellmate command line to the game window and the
// deal generator.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/thomaslaurenson/cellmate/internal/deal"
	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/store"
	"github.com/thomaslaurenson/cellmate/internal/ui"
)

// PlayFunc opens the game window and blocks until it is closed or ctx is
// cancelled. main passes RunGame; tests pass a fake so no window opens.
type PlayFunc func(ctx context.Context, opts ui.Options) error

// RunGame opens the real game window. It exists so main can hand the window
// to NewRootCmd without importing the ui package itself.
func RunGame(ctx context.Context, opts ui.Options) error {
	return ui.Run(ctx, opts)
}

// PathFunc returns where settings and statistics are kept. main passes
// DataPath; tests pass their own. An empty path keeps nothing.
type PathFunc func() (string, error)

// App holds the dependencies shared by every subcommand.
type App struct {
	play     PlayFunc
	dataPath PathFunc
}

// NewRootCmd builds the command tree, writing output to out and errw. A bare
// invocation opens the game through play, keeping settings and statistics in
// the file dataPath names. dataPath is only called when the game opens, so a
// subcommand never reports a problem with a file it does not use.
func NewRootCmd(out, errw io.Writer, play PlayFunc, dataPath PathFunc) *cobra.Command {
	a := &App{play: play, dataPath: dataPath}
	var editionName string
	var game int

	root := &cobra.Command{
		Use:           "cellmate",
		Short:         "FreeCell in the style of Windows 95 and XP",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       Version,
		RunE: func(cmd *cobra.Command, _ []string) error {
			saved, st := a.load(cmd.ErrOrStderr())
			ed, err := resolveEdition(editionName, cmd.Flags().Changed("edition"), saved.Settings.Edition)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("game") {
				if err := ed.CheckDeal(game); err != nil {
					return err
				}
			}
			return a.play(cmd.Context(), ui.Options{
				Edition: ed,
				Deal:    game,
				Saved:   saved,
				Store:   st,
				CanExit: true,
				Version: Version,
			})
		},
	}
	root.SetOut(out)
	root.SetErr(errw)

	root.PersistentFlags().StringVar(&editionName, "edition", edition.Default.String(),
		"edition of Windows to reproduce: 95 or xp")
	root.Flags().IntVar(&game, "game", 0, "deal number to start with instead of a random one")
	registerCompletions(root)

	root.AddCommand(newDealCmd(&editionName), newVersionCmd())
	return root
}

func newDealCmd(editionName *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deal <number>",
		Short: "Print the cards of a numbered deal, one row per line",
		Example: "  cellmate deal 617\n" +
			"  cellmate deal -- -1    # one of the two deals that cannot be won",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			ed, err := edition.Parse(*editionName)
			if err != nil {
				return err
			}
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("deal %q is not a whole number", args[0])
			}
			if err := ed.CheckDeal(n); err != nil {
				return err
			}
			return deal.WriteRows(cmd.OutOrStdout(), n)
		},
	}
	cmd.SetFlagErrorFunc(negativeDealHint)
	return cmd
}

// negativeDealHint explains the one flag error that is really a deal number.
// The impossible deals are -1 and -2, which the flag parser takes for
// shorthand flags, so "deal -1" fails before the command sees it. The hint
// is given only when the unknown shorthands read as a number, so a mistyped
// flag such as -x gets the parser's error alone.
func negativeDealHint(_ *cobra.Command, err error) error {
	var ne *pflag.NotExistError
	if !errors.As(err, &ne) {
		return err
	}
	n := "-" + ne.GetSpecifiedShortnames()
	if _, convErr := strconv.Atoi(n); convErr != nil {
		return err
	}
	return fmt.Errorf("%w (a negative deal goes after --, as in: cellmate deal -- %s)", err, n)
}

// load reads the saved settings and statistics. A file that cannot be read
// is reported and then left alone: the game runs with defaults and saves
// nothing, rather than overwriting data it failed to understand.
func (a *App) load(errw io.Writer) (store.Data, ui.Saver) {
	path, err := a.dataPath()
	if err != nil {
		fmt.Fprintf(errw, "[!] %v; playing without saving statistics\n", err)
		return store.Defaults(), nil
	}
	if path == "" {
		return store.Defaults(), nil
	}
	f := store.NewFile(path)
	d, err := f.Load()
	if err != nil {
		fmt.Fprintf(errw, "[!] %v; playing without saving statistics\n", err)
		return d, nil
	}
	return d, f
}

// resolveEdition picks the edition from the --edition flag when it was
// given, then the one saved in Options, then the default.
func resolveEdition(flag string, flagSet bool, saved string) (edition.Edition, error) {
	if flagSet || saved == "" {
		return edition.Parse(flag)
	}
	if ed, err := edition.Parse(saved); err == nil {
		return ed, nil
	}
	return edition.Default, nil
}

// DataPath returns where the desktop build keeps its settings and
// statistics: a cellmate directory in the user's configuration directory.
func DataPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find configuration directory: %w", err)
	}
	return filepath.Join(dir, "cellmate", "cellmate.json"), nil
}
