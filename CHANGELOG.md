# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
