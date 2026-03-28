package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nlink-jp/swrite/internal/slack"
	"github.com/spf13/cobra"
)

func newUploadCmd(flagQuiet *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a file to Slack",
		Long: `Upload a file to a Slack channel or DM.

The file can be read from a path (--file) or from stdin (--file -).
When reading from stdin, a temporary file is created automatically.

Examples:
  swrite upload -f report.csv -c "#data" --comment "Weekly report"
  cat output.log | swrite upload -f - -c "#ops" --filename "run.log"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpload(cmd, flagQuiet)
		},
	}

	cmd.Flags().StringP("channel", "c", "", "Destination channel (overrides profile default)")
	cmd.Flags().String("user", "", "Send as a DM to a user ID")
	cmd.Flags().StringP("file", "f", "", `File path to upload, or "-" to read from stdin`)
	cmd.Flags().StringP("comment", "m", "", "Initial comment to post with the file")
	cmd.Flags().StringP("filename", "n", "", "Filename shown in Slack (default: basename of --file)")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runUpload(cmd *cobra.Command, flagQuiet *bool) error {
	if state.profile.Token == "" {
		return fmt.Errorf("no token configured; run 'swrite config init' and 'swrite profile add'")
	}

	channel, _ := cmd.Flags().GetString("channel")
	userID, _ := cmd.Flags().GetString("user")
	filePath, _ := cmd.Flags().GetString("file")
	comment, _ := cmd.Flags().GetString("comment")
	filename, _ := cmd.Flags().GetString("filename")

	if channel != "" && userID != "" {
		return fmt.Errorf("--channel and --user are mutually exclusive")
	}

	if filePath == "-" {
		tmp, err := os.CreateTemp("", "swrite-stdin-*")
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}
		defer os.Remove(tmp.Name())

		if _, err := io.Copy(tmp, os.Stdin); err != nil {
			return fmt.Errorf("buffer stdin: %w", err)
		}
		if err := tmp.Close(); err != nil {
			return fmt.Errorf("close temp file: %w", err)
		}
		filePath = tmp.Name()
		if filename == "" {
			filename = "stdin"
		}
	} else {
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("file %q: %w", filePath, err)
		}
		if filename == "" {
			filename = filepath.Base(filePath)
		}
	}

	client := newClient(state.profile.Token, state.cacheDir)
	opts := slack.PostFileOptions{
		Channel:        channel,
		UserID:         userID,
		DefaultChannel: state.profile.Channel,
		FilePath:       filePath,
		Filename:       filename,
		Comment:        comment,
	}
	if err := client.PostFile(cmd.Context(), opts); err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	if !*flagQuiet {
		fmt.Fprintf(os.Stderr, "File %q uploaded.\n", filename)
	}
	return nil
}
