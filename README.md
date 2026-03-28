# swrite

Post messages and files to Slack from the command line.

Designed for bot workflows and shell pipelines. Works alongside [stail](https://github.com/nlink-jp/stail) and [slack-router](https://github.com/nlink-jp/slack-router).

## Features

- Post text or Block Kit messages to channels or DMs
- Stream stdin line by line, batching every 3 seconds (`--stream`)
- Upload files with an optional initial comment
- Profile-based configuration — switch between workspaces easily
- Server mode — zero config files via environment variables (Docker, Kubernetes)
- Auto-join on `not_in_channel` — no manual bot invitation required

## Setup

New to swrite? See the **[Slack Setup Guide](docs/SLACK_SETUP.md)** for step-by-step instructions on creating a Slack App and obtaining a bot token.

## Installation

Download the latest binary for your platform from [Releases](https://github.com/nlink-jp/swrite/releases),
or build from source:

```bash
make build   # outputs dist/swrite
```

## Quick Start

```bash
# 1. Create a config file
swrite config init

# 2. Add a profile (you will be prompted for the bot token)
swrite profile add myworkspace --channel "#general"

# 3. Post a message
echo "deploy complete" | swrite post -c "#ops"
```

## Commands

### `swrite post`

Post a text or Block Kit message.

```
swrite post [message text] [flags]

Flags:
  -c, --channel string     Destination channel (overrides profile default)
      --user string        Send as a DM to a user ID
  -f, --from-file string   Read message body from a file
  -s, --stream             Stream stdin line by line, batching every 3 seconds
  -t, --tee                Echo stdin to stdout before posting
  -u, --username string    Override display name for this post
  -i, --icon-emoji string  Override icon emoji (e.g. :robot_face:)
      --format string      Message format: text or blocks (default "text")
```

**Examples:**

```bash
# Plain text from argument
swrite post "hello world" -c "#general"

# From stdin
echo "server is up" | swrite post -c "#ops"

# Block Kit from file
cat payload.json | swrite post --format blocks -c "#alerts"

# Stream (useful with tail -f or slack-router)
tail -f /var/log/app.log | swrite post --stream -c "#logs"
```

### `swrite upload`

Upload a file to Slack.

```
swrite upload [flags]

Flags:
  -c, --channel string    Destination channel (overrides profile default)
      --user string       Send as a DM to a user ID
  -f, --file string       File path to upload, or "-" for stdin (required)
  -m, --comment string    Initial comment to post with the file
  -n, --filename string   Filename shown in Slack (default: basename of --file)
```

**Examples:**

```bash
swrite upload -f report.csv -c "#data" --comment "Weekly report"
cat output.log | swrite upload -f - -c "#ops" --filename "run.log"
```

### `swrite config init`

Create a default config file at `~/.config/swrite/config.json`.

### `swrite cache`

swrite caches the channel list for the active profile to avoid repeated API calls
on every invocation. The cache is stored in `~/.config/swrite/cache/<profile>/` and
expires after one hour.

```bash
swrite cache clear   # delete cached data for the active profile
```

### `swrite profile`

Manage named profiles.

```bash
swrite profile add myworkspace --channel "#general"   # prompts for token
swrite profile list
swrite profile use myworkspace
swrite profile set channel "#ops"
swrite profile set token                              # secure prompt
swrite profile remove old-workspace
```

## Configuration

The config file lives at `~/.config/swrite/config.json` (0600 permissions).
The schema is compatible with [stail](https://github.com/nlink-jp/stail).

```json
{
  "current_profile": "production",
  "profiles": {
    "production": {
      "provider": "slack",
      "token": "xoxb-YOUR-BOT-TOKEN",
      "channel": "#alerts",
      "username": "mybot"
    },
    "staging": {
      "provider": "slack",
      "token": "xoxb-STAGING-TOKEN",
      "channel": "#staging-logs"
    }
  }
}
```

### Profile fields

| Field | Required | Description |
|---|---|---|
| `provider` | yes | Always `slack` |
| `token` | yes | Slack Bot Token (`xoxb-...`) |
| `channel` | no | Default destination channel (`#name` or ID) |
| `username` | no | Default display name override |

## Server Mode

Set `SWRITE_MODE=server` to skip the config file entirely. Required when running as a bot in Docker or Kubernetes.

| Variable | Required | Description |
|---|---|---|
| `SWRITE_MODE` | yes | Must be `server` |
| `SWRITE_TOKEN` | yes | Slack Bot Token |
| `SWRITE_CHANNEL` | no | Default channel |
| `SWRITE_USERNAME` | no | Default display name |
| `SWRITE_CACHE_DIR` | no | Directory to cache the channel list (recommended when swrite is invoked repeatedly) |

**Example (Docker):**

```bash
docker run --rm \
  -e SWRITE_MODE=server \
  -e SWRITE_TOKEN=xoxb-... \
  -e SWRITE_CHANNEL="#alerts" \
  myimage sh -c 'echo "container started" | swrite post'
```

## Required Slack Scopes

| Scope | Used by |
|---|---|
| `chat:write` | `post` |
| `files:write` | `upload` |
| `channels:read` | channel name resolution |
| `groups:read` | private channel name resolution |
| `channels:join` | auto-join on `not_in_channel` |
| `im:write` | DM via `--user` |

## Global Flags

```
  --config string    Config file path (default: ~/.config/swrite/config.json)
  -p, --profile string  Profile to use (overrides current profile)
  -q, --quiet           Suppress informational messages on stderr
```
