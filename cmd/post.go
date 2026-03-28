package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/nlink-jp/swrite/internal/slack"
	"github.com/spf13/cobra"
)

func newPostCmd(flagQuiet *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post [message text]",
		Short: "Post a message to Slack",
		Long: `Post a text or Block Kit message to a Slack channel or DM.

Message content is sourced in order: argument → --from-file → stdin.

Examples:
  swrite post "hello world" -c "#general"
  echo "deploy done" | swrite post -c "#ops"
  cat blocks.json | swrite post --format blocks -c "#alerts"
  tail -f app.log | swrite post --stream -c "#logs"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPost(cmd, args, flagQuiet)
		},
	}

	cmd.Flags().StringP("channel", "c", "", "Destination channel (overrides profile default)")
	cmd.Flags().String("user", "", "Send as a DM to a user ID")
	cmd.Flags().StringP("from-file", "f", "", "Read message body from a file")
	cmd.Flags().BoolP("stream", "s", false, "Stream stdin line by line, batching every 3 seconds")
	cmd.Flags().BoolP("tee", "t", false, "Echo stdin to stdout before posting (stream and stdin modes)")
	cmd.Flags().StringP("username", "u", "", "Override display name for this post")
	cmd.Flags().StringP("icon-emoji", "i", "", "Override icon emoji for this post (e.g. :robot_face:)")
	cmd.Flags().String("format", "text", "Message format: text or blocks")

	return cmd
}

func runPost(cmd *cobra.Command, args []string, flagQuiet *bool) error {
	if state.profile.Token == "" {
		return fmt.Errorf("no token configured; run 'swrite config init' and 'swrite profile add'")
	}

	channel, _ := cmd.Flags().GetString("channel")
	userID, _ := cmd.Flags().GetString("user")
	fromFile, _ := cmd.Flags().GetString("from-file")
	stream, _ := cmd.Flags().GetBool("stream")
	tee, _ := cmd.Flags().GetBool("tee")
	username, _ := cmd.Flags().GetString("username")
	iconEmoji, _ := cmd.Flags().GetString("icon-emoji")
	format, _ := cmd.Flags().GetString("format")

	if channel != "" && userID != "" {
		return fmt.Errorf("--channel and --user are mutually exclusive")
	}
	if format != "text" && format != "blocks" {
		return fmt.Errorf("--format must be \"text\" or \"blocks\", got %q", format)
	}
	if stream && format == "blocks" {
		return fmt.Errorf("--stream cannot be used with --format blocks")
	}

	if username == "" {
		username = state.profile.Username
	}

	client := newClient(state.profile.Token)

	if stream {
		return runStream(cmd.Context(), client, channel, userID, username, iconEmoji, tee, flagQuiet)
	}

	var content string
	switch {
	case len(args) > 0:
		content = strings.Join(args, " ")
	case fromFile != "":
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return fmt.Errorf("read file %s: %w", fromFile, err)
		}
		content = string(data)
	default:
		stat, _ := os.Stdin.Stat()
		if stat.Mode()&os.ModeCharDevice != 0 {
			return fmt.Errorf("no message provided: use an argument, --from-file, or pipe from stdin")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		content = string(data)
		if tee {
			fmt.Print(content)
		}
	}

	opts := slack.PostMessageOptions{
		Channel:        channel,
		UserID:         userID,
		DefaultChannel: state.profile.Channel,
		Username:       username,
		IconEmoji:      iconEmoji,
	}

	if format == "blocks" {
		blocks, err := parseBlocks(content)
		if err != nil {
			return err
		}
		opts.Blocks = blocks
	} else {
		opts.Text = content
	}

	if err := client.PostMessage(cmd.Context(), opts); err != nil {
		return fmt.Errorf("post message: %w", err)
	}
	if !*flagQuiet {
		fmt.Fprintln(os.Stderr, "Message posted.")
	}
	return nil
}

// parseBlocks accepts either a {"blocks":[...]} wrapper or a bare [...] array.
func parseBlocks(content string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "[") {
		var arr []any
		if err := json.Unmarshal([]byte(content), &arr); err != nil {
			return nil, fmt.Errorf("parse blocks array: %w", err)
		}
		return json.RawMessage(content), nil
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &wrapper); err != nil {
		return nil, fmt.Errorf("parse blocks JSON: %w", err)
	}
	blocks, ok := wrapper["blocks"]
	if !ok {
		return nil, fmt.Errorf("blocks JSON must be an array or an object with a \"blocks\" key")
	}
	return blocks, nil
}

// runStream reads stdin line by line, batching messages every 3 seconds.
func runStream(ctx context.Context, client slack.Client, channel, userID, username, iconEmoji string, tee bool, flagQuiet *bool) error {
	if !*flagQuiet {
		fmt.Fprintln(os.Stderr, "Streaming stdin to Slack. Press Ctrl+C to stop.")
	}

	lines := make(chan string)
	scanner := bufio.NewScanner(os.Stdin)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if tee {
				fmt.Println(line)
			}
			lines <- line
		}
		close(lines)
	}()

	var buffer []string
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	flush := func() {
		if len(buffer) == 0 {
			return
		}
		opts := slack.PostMessageOptions{
			Channel:        channel,
			UserID:         userID,
			DefaultChannel: state.profile.Channel,
			Text:           strings.Join(buffer, "\n"),
			Username:       username,
			IconEmoji:      iconEmoji,
		}
		if err := client.PostMessage(ctx, opts); err != nil {
			fmt.Fprintf(os.Stderr, "post error: %v\n", err)
		} else if !*flagQuiet {
			fmt.Fprintf(os.Stderr, "Posted %d line(s).\n", len(buffer))
		}
		buffer = nil
	}

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				flush()
				if !*flagQuiet {
					fmt.Fprintln(os.Stderr, "Stream finished.")
				}
				return nil
			}
			buffer = append(buffer, line)
		case <-ticker.C:
			flush()
		}
	}
}
