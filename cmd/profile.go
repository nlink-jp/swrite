package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/nlink-jp/swrite/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newProfileCmd() *cobra.Command {
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage configuration profiles",
	}

	// ── profile list ───────────────────────────────────────────────────────────
	profileCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := requireCLIMode(); err != nil {
				return err
			}
			cfg, err := loadConfig(state.configPath)
			if err != nil {
				return err
			}
			for name, p := range cfg.Profiles {
				marker := "  "
				if name == cfg.CurrentProfile {
					marker = "* "
				}
				fmt.Printf("%s%s (provider: %s)\n", marker, name, p.Provider)
			}
			return nil
		},
	})

	// ── profile use ────────────────────────────────────────────────────────────
	profileCmd.AddCommand(&cobra.Command{
		Use:   "use <profile-name>",
		Short: "Set the active profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := requireCLIMode(); err != nil {
				return err
			}
			name := args[0]
			cfg, err := loadConfig(state.configPath)
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("profile %q not found", name)
			}
			cfg.CurrentProfile = name
			if err := config.Save(cfg, state.configPath); err != nil {
				return err
			}
			fmt.Printf("Active profile set to %q\n", name)
			return nil
		},
	})

	// ── profile add ────────────────────────────────────────────────────────────
	addCmd := &cobra.Command{
		Use:   "add <profile-name>",
		Short: "Add a new profile",
		Long: `Add a new profile. You will be prompted to enter the Bot Token securely
(input is not echoed).

Example:
  swrite profile add myworkspace --provider slack --channel "#general"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireCLIMode(); err != nil {
				return err
			}
			name := args[0]
			cfg, err := loadConfig(state.configPath)
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[name]; ok {
				return fmt.Errorf("profile %q already exists", name)
			}

			token, err := readSecret("Bot Token (xoxb-...): ")
			if err != nil {
				return fmt.Errorf("read token: %w", err)
			}

			provider, _ := cmd.Flags().GetString("provider")
			channel, _ := cmd.Flags().GetString("channel")
			username, _ := cmd.Flags().GetString("username")

			cfg.Profiles[name] = config.Profile{
				Provider: provider,
				Token:    token,
				Channel:  channel,
				Username: username,
			}
			if err := config.Save(cfg, state.configPath); err != nil {
				return err
			}
			fmt.Printf("Profile %q added.\n", name)
			return nil
		},
	}
	addCmd.Flags().String("provider", config.ProviderSlack, "provider type")
	addCmd.Flags().StringP("channel", "c", "", "default channel")
	addCmd.Flags().StringP("username", "u", "", "default display name")
	profileCmd.AddCommand(addCmd)

	// ── profile set ────────────────────────────────────────────────────────────
	profileCmd.AddCommand(&cobra.Command{
		Use:   "set <key> [value]",
		Short: "Set a field in the active profile",
		Long: `Set a field in the currently active profile.

Settable keys:
  provider   Provider type (slack)
  token      Bot Token (prompted securely if value is omitted)
  channel    Default channel
  username   Default display name

Example:
  swrite profile set channel "#ops"
  swrite profile set token`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := requireCLIMode(); err != nil {
				return err
			}
			key := args[0]
			cfg, err := loadConfig(state.configPath)
			if err != nil {
				return err
			}
			name := cfg.CurrentProfile
			p, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("current profile %q not found", name)
			}

			value := ""
			if len(args) == 2 {
				value = args[1]
			}

			switch key {
			case "provider":
				if value == "" {
					return fmt.Errorf("provider requires a value")
				}
				p.Provider = value
			case "token":
				if value == "" {
					value, err = readSecret("Bot Token: ")
					if err != nil {
						return err
					}
				}
				p.Token = value
			case "channel":
				p.Channel = value
			case "username":
				p.Username = value
			default:
				return fmt.Errorf("unknown key %q — valid keys: provider, token, channel, username", key)
			}

			cfg.Profiles[name] = p
			if err := config.Save(cfg, state.configPath); err != nil {
				return err
			}
			fmt.Printf("Profile %q updated: %s\n", name, key)
			return nil
		},
	})

	// ── profile remove ─────────────────────────────────────────────────────────
	removeCmd := &cobra.Command{
		Use:     "remove <profile-name>",
		Aliases: []string{"rm"},
		Short:   "Remove a profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := requireCLIMode(); err != nil {
				return err
			}
			name := args[0]
			if name == "default" {
				return fmt.Errorf("cannot remove the default profile")
			}
			cfg, err := loadConfig(state.configPath)
			if err != nil {
				return err
			}
			if name == cfg.CurrentProfile {
				return fmt.Errorf("cannot remove the currently active profile %q", name)
			}
			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("profile %q not found", name)
			}
			delete(cfg.Profiles, name)
			if err := config.Save(cfg, state.configPath); err != nil {
				return err
			}
			fmt.Printf("Profile %q removed.\n", name)
			return nil
		},
	}
	profileCmd.AddCommand(removeCmd)

	return profileCmd
}

// readSecret reads a secret from the terminal without echoing input.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
