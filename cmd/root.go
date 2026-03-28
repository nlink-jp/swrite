// Package cmd implements the swrite CLI commands.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/nlink-jp/swrite/internal/config"
	"github.com/spf13/cobra"
)

// appState holds runtime configuration shared across subcommands.
type appState struct {
	serverMode  bool
	profile     config.Profile
	configPath  string
	profileName string
	cacheDir    string
}

// state is populated by persistentPreRunE before any subcommand runs.
var state appState

// newRootCmd builds and returns a fresh command tree.
// A fresh tree ensures flag values do not persist between Execute calls (important for tests).
func newRootCmd() *cobra.Command {
	var flagConfig, flagProfile string
	var flagQuiet bool

	root := &cobra.Command{
		Use:   "swrite",
		Short: "Slack writer — post messages and files to Slack",
		Long: `swrite posts messages and files to Slack from the command line.

Designed for bot workflows and shell pipelines. Works alongside stail and slack-router.

Examples:
  echo "deploy complete" | swrite post -c "#ops"
  swrite post --format blocks < payload.json
  swrite upload -f report.csv -c "#data" --comment "Weekly report"`,
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&flagConfig, "config", "",
		"config file path (default: ~/.config/swrite/config.json)")
	root.PersistentFlags().StringVarP(&flagProfile, "profile", "p", "",
		"profile to use (overrides current profile)")
	root.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false,
		"suppress informational messages on stderr")

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if skipConfigLoad(cmd) {
			return nil
		}

		serverMode, err := config.DetectServerMode()
		if err != nil {
			return err
		}
		state.serverMode = serverMode

		if serverMode {
			if flagConfig != "" {
				return fmt.Errorf("--config flag cannot be used in server mode (SWRITE_MODE=server)")
			}
			if flagProfile != "" {
				return fmt.Errorf("--profile flag cannot be used in server mode (SWRITE_MODE=server)")
			}
			cfg, err := config.BuildConfigFromEnv()
			if err != nil {
				return err
			}
			p, err := cfg.GetProfile("")
			if err != nil {
				return err
			}
			state.profile = p
			state.cacheDir = config.ServerCacheDir()
			return nil
		}

		cfgPath := flagConfig
		if cfgPath == "" {
			cfgPath, err = config.DefaultConfigPath()
			if err != nil {
				return fmt.Errorf("resolve config path: %w", err)
			}
		}
		state.configPath = cfgPath

		cfg, err := config.Load(cfgPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}

		profileName := flagProfile
		if profileName == "" {
			profileName = cfg.CurrentProfile
		}
		p, err := cfg.GetProfile(flagProfile)
		if err != nil {
			return err
		}
		state.profile = p
		state.profileName = profileName
		state.cacheDir = config.DefaultCacheDir(profileName)
		return nil
	}

	root.AddCommand(newPostCmd(&flagQuiet))
	root.AddCommand(newUploadCmd(&flagQuiet))
	root.AddCommand(newConfigCmd())
	root.AddCommand(newProfileCmd())
	root.AddCommand(newCacheCmd())

	return root
}

// RootCmd returns a fresh root command (used in tests).
func RootCmd() *cobra.Command {
	return newRootCmd()
}

// Execute runs the CLI. Call this from main().
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// skipConfigLoad returns true for commands that manage the config file directly.
func skipConfigLoad(cmd *cobra.Command) bool {
	c := cmd
	for c != nil {
		name := c.Name()
		if name == "config" || name == "profile" {
			return true
		}
		c = c.Parent()
	}
	return false
}

// requireCLIMode returns an error when run in server mode.
func requireCLIMode() error {
	if state.serverMode {
		return fmt.Errorf("this command is not available in server mode (SWRITE_MODE=server)")
	}
	return nil
}

// loadConfig loads the current config from disk. Must only be called in CLI mode.
func loadConfig(configPath string) (*config.Config, error) {
	if configPath == "" {
		var err error
		configPath, err = config.DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}
	return config.Load(configPath)
}
