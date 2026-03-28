package cmd

import (
	"fmt"
	"os"

	"github.com/nlink-jp/swrite/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	configCmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create a default configuration file",
		Long: `Create a default configuration file at ~/.config/swrite/config.json.

The file is created with 0600 permissions (owner read/write only).
This command is not available in server mode (SWRITE_MODE=server).

Example:
  swrite config init`,
		RunE: runConfigInit,
	})

	return configCmd
}

func runConfigInit(_ *cobra.Command, _ []string) error {
	if err := requireCLIMode(); err != nil {
		return err
	}

	cfgPath := state.configPath
	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultConfigPath()
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("config file already exists at %s", cfgPath)
	}

	cfg := config.DefaultConfig()
	if err := config.Save(cfg, cfgPath); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Config file created: %s\n", cfgPath)
	fmt.Fprintln(os.Stdout, "Add a profile with: swrite profile add <name>")
	return nil
}
