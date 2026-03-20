package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCommand(rootCmd *cobra.Command) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
	}

	completionCmd.AddCommand(&cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		RunE: func(_ *cobra.Command, _ []string) error {
			return rootCmd.GenFishCompletion(os.Stdout, true)
		},
	})
	return completionCmd
}
