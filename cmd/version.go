package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCmd prints the same line as --version. Neither reads a config
// file: cobra answers --version before any pre-run hook, and skipConfigLoad
// passes this command through.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := cmd.Root()
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", root.Name(), root.Version)
			return err
		},
	}
}
