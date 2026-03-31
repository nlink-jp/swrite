// Package slack provides a write-only Slack Web API client for swrite.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultBaseURL  = "https://slack.com/api"
	httpTimeout     = 30 * time.Second
	channelCacheTTL = time.Hour
)

// channelEntry is a minimal channel record used for name-to-ID resolution.
type channelEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PostResult holds the result of a successful chat.postMessage call.
type PostResult struct {
	TS      string `json:"ts"`
	Channel string `json:"channel"`
}

// Client defines the write operations used by swrite.
type Client interface {
	PostMessage(ctx context.Context, opts PostMessageOptions) (PostResult, error)
	PostFile(ctx context.Context, opts PostFileOptions) error
}

// PostMessageOptions holds parameters for chat.postMessage.
type PostMessageOptions struct {
	Channel        string          // explicit --channel flag value (name or ID)
	UserID         string          // explicit --user flag value; opens a DM
	DefaultChannel string          // profile.Channel fallback
	Text           string
	Username       string
	IconEmoji      string
	Blocks         json.RawMessage
	Attachments    json.RawMessage
	UnfurlLinks    *bool // nil = Slack default (true); set to disable link previews
	UnfurlMedia    *bool
}

// PostFileOptions holds parameters for the external file upload flow.
type PostFileOptions struct {
	Channel        string
	UserID         string
	DefaultChannel string
	FilePath       string
	Filename       string
	Comment        string
	ThreadTS       string // post file as a thread reply
}

// HTTPClient is the production implementation of Client.
type HTTPClient struct {
	token      string
	baseURL    string
	httpClient *http.Client
	cacheDir   string // optional; empty = no disk cache
}

// NewHTTPClient creates a new HTTPClient with the given bot token.
func NewHTTPClient(token string) *HTTPClient {
	return &HTTPClient{
		token:   token,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// WithBaseURL overrides the API base URL (used in tests).
func (c *HTTPClient) WithBaseURL(u string) *HTTPClient {
	c.baseURL = strings.TrimRight(u, "/")
	return c
}

// SetCacheDir enables disk caching of the channel list for name-to-ID resolution.
// dir is profile-specific and is created on first use.
func (c *HTTPClient) SetCacheDir(dir string) {
	c.cacheDir = dir
}

// apiResponse is the common Slack envelope.
type apiResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// do executes an HTTP request to the given full URL.
// It checks HTTP-level errors and the Slack ok:false envelope.
func (c *HTTPClient) do(ctx context.Context, method, fullURL string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http %s: %w", method, err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}

	var base apiResponse
	if jerr := json.Unmarshal(b, &base); jerr == nil && !base.OK {
		return nil, fmt.Errorf("slack API error: %s", base.Error)
	}
	return b, nil
}

// apiGet sends a GET request to the Slack API.
func (c *HTTPClient) apiGet(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	u := fmt.Sprintf("%s/%s?%s", c.baseURL, endpoint, params.Encode())
	return c.do(ctx, http.MethodGet, u, nil, "")
}

// apiPost sends a POST request with a JSON body to the Slack API.
func (c *HTTPClient) apiPost(ctx context.Context, endpoint string, payload any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	u := fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	return c.do(ctx, http.MethodPost, u, bytes.NewReader(b), "application/json; charset=utf-8")
}

// resolveTarget returns a Slack channel ID from the provided target options.
func (c *HTTPClient) resolveTarget(ctx context.Context, channel, userID, defaultChannel string) (string, error) {
	switch {
	case userID != "":
		return c.openDM(ctx, userID)
	case channel != "":
		return c.resolveChannelID(ctx, channel)
	case defaultChannel != "":
		return c.resolveChannelID(ctx, defaultChannel)
	default:
		return "", fmt.Errorf("no destination: set a channel in the profile or use --channel/--user")
	}
}

// resolveChannelID converts a channel name or ID to a Slack channel ID.
func (c *HTTPClient) resolveChannelID(ctx context.Context, nameOrID string) (string, error) {
	name := strings.TrimPrefix(nameOrID, "#")
	// Slack IDs start with C, G, or D and are at least 9 chars.
	if len(name) >= 9 && (name[0] == 'C' || name[0] == 'G' || name[0] == 'D') {
		return name, nil
	}

	channels, err := c.listChannels(ctx)
	if err != nil {
		return "", err
	}
	for _, ch := range channels {
		if ch.Name == name || ch.ID == name {
			return ch.ID, nil
		}
	}
	return "", fmt.Errorf("channel %q not found", nameOrID)
}

// listChannels returns all channels, using a disk cache when available.
func (c *HTTPClient) listChannels(ctx context.Context) ([]channelEntry, error) {
	if c.cacheDir != "" {
		cachePath := filepath.Join(c.cacheDir, "channels.json")
		if entries, ok := loadCache[[]channelEntry](cachePath, channelCacheTTL); ok {
			return entries, nil
		}
	}

	var all []channelEntry
	cursor := ""
	for {
		params := url.Values{
			"limit":            {"200"},
			"exclude_archived": {"true"},
			"types":            {"public_channel,private_channel"},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		b, err := c.apiGet(ctx, "conversations.list", params)
		if err != nil {
			return nil, fmt.Errorf("list channels: %w", err)
		}
		var resp struct {
			Channels         []channelEntry `json:"channels"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		if err := json.Unmarshal(b, &resp); err != nil {
			return nil, fmt.Errorf("parse channels: %w", err)
		}
		all = append(all, resp.Channels...)
		cursor = resp.ResponseMetadata.NextCursor
		if cursor == "" {
			break
		}
	}

	if c.cacheDir != "" {
		saveCache(filepath.Join(c.cacheDir, "channels.json"), all)
	}
	return all, nil
}

// openDM opens a direct message channel with userID and returns the channel ID.
func (c *HTTPClient) openDM(ctx context.Context, userID string) (string, error) {
	b, err := c.apiPost(ctx, "conversations.open", map[string]string{"users": userID})
	if err != nil {
		return "", fmt.Errorf("conversations.open: %w", err)
	}
	var resp struct {
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", fmt.Errorf("parse conversations.open: %w", err)
	}
	if resp.Channel.ID == "" {
		return "", fmt.Errorf("conversations.open: no channel ID returned")
	}
	return resp.Channel.ID, nil
}

// joinChannel joins a public channel (used for not_in_channel retry).
func (c *HTTPClient) joinChannel(ctx context.Context, channelID string) error {
	_, err := c.apiPost(ctx, "conversations.join", map[string]string{"channel": channelID})
	return err
}

// PostMessage sends a text or Block Kit message to a channel or DM.
// If the bot is not in the channel, it joins automatically and retries once.
func (c *HTTPClient) PostMessage(ctx context.Context, opts PostMessageOptions) (PostResult, error) {
	channelID, err := c.resolveTarget(ctx, opts.Channel, opts.UserID, opts.DefaultChannel)
	if err != nil {
		return PostResult{}, err
	}

	type payload struct {
		Channel     string          `json:"channel"`
		Text        string          `json:"text,omitempty"`
		Username    string          `json:"username,omitempty"`
		IconEmoji   string          `json:"icon_emoji,omitempty"`
		Blocks      json.RawMessage `json:"blocks,omitempty"`
		Attachments json.RawMessage `json:"attachments,omitempty"`
		UnfurlLinks *bool           `json:"unfurl_links,omitempty"`
		UnfurlMedia *bool           `json:"unfurl_media,omitempty"`
	}
	msg := payload{
		Channel:     channelID,
		Text:        opts.Text,
		Username:    opts.Username,
		IconEmoji:   opts.IconEmoji,
		Blocks:      opts.Blocks,
		Attachments: opts.Attachments,
		UnfurlLinks: opts.UnfurlLinks,
		UnfurlMedia: opts.UnfurlMedia,
	}

	body, err := c.apiPost(ctx, "chat.postMessage", msg)
	if err != nil && strings.Contains(err.Error(), "not_in_channel") && opts.UserID == "" {
		if jerr := c.joinChannel(ctx, channelID); jerr != nil {
			return PostResult{}, fmt.Errorf("join channel: %w", jerr)
		}
		body, err = c.apiPost(ctx, "chat.postMessage", msg)
	}
	if err != nil {
		return PostResult{}, err
	}

	var resp struct {
		TS      string `json:"ts"`
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return PostResult{TS: "", Channel: channelID}, nil
	}
	return PostResult{TS: resp.TS, Channel: resp.Channel}, nil
}

// PostFile uploads a file using the Slack external upload API (3 steps).
// If the bot is not in the channel, it joins automatically and retries step 3 once.
func (c *HTTPClient) PostFile(ctx context.Context, opts PostFileOptions) error {
	channelID, err := c.resolveTarget(ctx, opts.Channel, opts.UserID, opts.DefaultChannel)
	if err != nil {
		return err
	}

	// Step 1: Get an upload URL.
	fi, err := os.Stat(opts.FilePath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}
	params := url.Values{
		"filename": {opts.Filename},
		"length":   {fmt.Sprintf("%d", fi.Size())},
	}
	b, err := c.apiGet(ctx, "files.getUploadURLExternal", params)
	if err != nil {
		return fmt.Errorf("getUploadURLExternal: %w", err)
	}
	var urlResp struct {
		UploadURL string `json:"upload_url"`
		FileID    string `json:"file_id"`
	}
	if err := json.Unmarshal(b, &urlResp); err != nil {
		return fmt.Errorf("parse upload URL response: %w", err)
	}

	// Step 2: Upload the file to the provided URL (no Authorization header).
	f, err := os.Open(opts.FilePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPost, urlResp.UploadURL, f)
	if err != nil {
		return fmt.Errorf("build upload request: %w", err)
	}
	uploadReq.Header.Set("Content-Type", "application/octet-stream")
	uploadResp, err := c.httpClient.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	defer uploadResp.Body.Close()
	if uploadResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(uploadResp.Body)
		return fmt.Errorf("upload file: HTTP %d: %s", uploadResp.StatusCode, string(body))
	}

	// Step 3: Complete the upload.
	type fileEntry struct {
		ID string `json:"id"`
	}
	type completePayload struct {
		Files          []fileEntry `json:"files"`
		ChannelID      string      `json:"channel_id"`
		InitialComment string      `json:"initial_comment,omitempty"`
		ThreadTS       string      `json:"thread_ts,omitempty"`
	}
	cp := completePayload{
		Files:          []fileEntry{{ID: urlResp.FileID}},
		ChannelID:      channelID,
		InitialComment: opts.Comment,
		ThreadTS:       opts.ThreadTS,
	}
	_, err = c.apiPost(ctx, "files.completeUploadExternal", cp)
	if err != nil && strings.Contains(err.Error(), "not_in_channel") && opts.UserID == "" {
		if jerr := c.joinChannel(ctx, channelID); jerr != nil {
			return fmt.Errorf("join channel: %w", jerr)
		}
		_, err = c.apiPost(ctx, "files.completeUploadExternal", cp)
	}
	return err
}
