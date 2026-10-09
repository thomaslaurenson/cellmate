package cmd

import (
	"github.com/spf13/cobra"

	"github.com/thomaslaurenson/cellmate/internal/edition"
)

// registerCompletions offers the edition names for --edition, and nothing
// for --game, whose deal number a shell would otherwise complete as a file
// name.
func registerCompletions(root *cobra.Command) {
	// Registration fails only for a flag that does not exist, and the root
	// command defines both flags before calling this.
	_ = root.RegisterFlagCompletionFunc("edition", completeEditions)
	_ = root.RegisterFlagCompletionFunc("game", cobra.NoFileCompletions)
}

func completeEditions(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	names := make([]string, len(edition.All))
	for i, e := range edition.All {
		names[i] = e.String()
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
