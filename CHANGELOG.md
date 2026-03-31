# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.3] - 2026-03-31

### Fixed
- PostResult JSON parse failure now logs warning instead of silently swallowing error
- Test mock servers return ts/channel in responses for realistic testing

### Added
- Tests for ts stdout output after post
- Tests for upload --thread (thread_ts in API payload)

## [0.3.2] - 2026-03-31

### Added
- `upload --thread` flag for posting files as thread replies
- `PostFileOptions.ThreadTS` passes `thread_ts` to `files.completeUploadExternal`

## [0.3.1] - 2026-03-31

### Added
- `post` now outputs message timestamp (ts) to stdout for pipeline use
- Enables thread replies and file attachments to the posted message
- Human-readable confirmation on stderr (suppressible with `--quiet`)

## [0.3.0] - 2026-03-29

### Added

- **`--format payload`** — New message format that accepts a full Slack message JSON with `text`, `blocks`, and/or `attachments` fields. Enables colored sidebar panels via Slack attachments.
- **`--no-unfurl`** — Disable Slack link previews (`unfurl_links=false`, `unfurl_media=false`). Useful when posting messages with many URLs.
- **`unfurl_links` / `unfurl_media` in payload format** — When using `--format payload`, these fields are read from the JSON input and passed through to the Slack API.

## [0.2.0] - 2026-03-29

### Added

- **Channel list cache** — Channel name-to-ID resolution now caches `conversations.list` results on disk (TTL: 1 hour), avoiding repeated API calls on every invocation.
  - CLI mode: cache stored at `~/.config/swrite/cache/<profile>/channels.json`.
  - Server mode: cache enabled when `SWRITE_CACHE_DIR` is set (recommended for containers that run swrite repeatedly).
- **`swrite cache clear`** — Delete the cached channel data for the active profile.
- **Slack Setup Guide** — `docs/SLACK_SETUP.md` and `docs/SLACK_SETUP.ja.md` with step-by-step instructions for creating a Slack App.

## [0.1.0] - 2026-03-29

### Added

- **`swrite post`**: Post text or Block Kit messages to a Slack channel or DM.
  - Reads content from argument, `--from-file`, or stdin.
  - `--format blocks`: accepts a JSON array or `{"blocks":[...]}` wrapper.
  - `--stream`: batch-posts stdin lines every 3 seconds (for `tail -f` pipelines).
  - `--tee`: echoes stdin to stdout before posting.
  - `--channel` / `--user`: override destination per invocation.
  - `--username` / `--icon-emoji`: override bot display name and icon.
  - Auto-joins channel on `not_in_channel` error and retries.
- **`swrite upload`**: Upload a file to a Slack channel or DM.
  - Reads from a file path or stdin (`--file -`).
  - `--comment`: attach an initial comment.
  - `--filename`: override the filename shown in Slack.
  - Uses the Slack external upload API (3-step flow).
  - Auto-joins channel on `not_in_channel` error and retries.
- **`swrite config init`**: Create `~/.config/swrite/config.json` with a default profile.
- **`swrite profile add/list/use/set/remove`**: Manage named profiles (token, channel, username).
- **Server mode**: Set `SWRITE_MODE=server` to bypass config file and read `SWRITE_TOKEN`,
  `SWRITE_CHANNEL`, and `SWRITE_USERNAME` from environment variables.
  Designed for bot deployments (Docker, Kubernetes, slack-router).
- Profile config: `~/.config/swrite/config.json`, schema compatible with stail.
