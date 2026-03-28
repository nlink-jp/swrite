package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cache := &cobra.Command{
		Use:   "cache",
		Short: "Manage the local disk cache",
	}
	cache.AddCommand(newCacheClearCmd())
	return cache
}

func newCacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Delete cached channel data for the active profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireCLIMode(); err != nil {
				return err
			}
			dir := state.cacheDir
			if dir == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "no cache directory configured")
				return nil
			}
			if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("clear cache: %w", err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "cache cleared: %s\n", dir)
			return nil
		},
	}
}
