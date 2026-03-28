# swrite Slack Setup Guide

To use `swrite`, you need to create a Slack App and obtain a **Bot Token** (`xoxb-`).
Unlike stail, swrite does **not** require an App-Level Token or Socket Mode.

---

## Step 1: Create a Slack App

1. Go to the [Slack API site](https://api.slack.com/apps) and log in.

2. Click **"Create New App"**.

3. Select **"From scratch"**.

4. Enter an app name (e.g., `swrite`), select your workspace, and click **"Create App"**.

### Suggested Display Information

#### Short description

```
Post messages and files to Slack from the command line.
```

#### Long description

```
swrite is a bot-oriented command-line tool for posting to Slack. It sends text,
Block Kit messages, and files to channels or DMs from shell pipelines. Designed
for ChatOps workflows, alert notifications, and automated reporting.
```

---

## Step 2: Add Bot Token Scopes

1. From the left sidebar, click **"OAuth & Permissions"**.

2. Scroll to the **"Scopes"** section.

3. Under **"Bot Token Scopes"**, add the following scopes:

   **Required:**

   | Scope | Purpose |
   |---|---|
   | `chat:write` | Post messages (`swrite post`) |
   | `files:write` | Upload files (`swrite upload`) |
   | `channels:read` | Resolve public channel names to IDs |

   **Recommended:**

   | Scope | Purpose |
   |---|---|
   | `groups:read` | Resolve private channel names to IDs |
   | `channels:join` | Auto-join public channels on `not_in_channel` (avoids manual `/invite`) |
   | `im:write` | Send direct messages (`--user` flag) |

> **Note:** If you skip `channels:join`, you must manually invite the bot to every
> public channel with `/invite @<app-name>` before posting.

---

## Step 3: Install the App to Your Workspace

1. From the left sidebar, click **"OAuth & Permissions"**.

2. Click **"Install to Workspace"** at the top of the page.

3. Click **"Allow"** to authorize.

---

## Step 4: Copy Your Bot Token

After installation, the **"OAuth & Permissions"** page shows a
**"Bot User OAuth Token"** starting with `xoxb-`. Copy it — you will need it in the next step.

---

## Step 5: Configure swrite

### Initialize the config file

```bash
swrite config init
```

This creates `~/.config/swrite/config.json` with `0600` permissions.

### Add a profile

```bash
swrite profile add my-workspace --channel "#alerts"
```

You will be prompted for the bot token:

```
Bot Token (xoxb-...): [paste your xoxb- token]
```

### Set the active profile

```bash
swrite profile use my-workspace
```

Your setup is complete.

---

## Step 6: Invite the Bot to Private Channels

For **private channels**, the bot must be invited manually.
In each private channel you want to post to, type:

```
/invite @<your-app-name>
```

For **public channels**, the bot joins automatically on first use (if `channels:join` scope is granted).

---

## Verification

Test that everything is working:

```bash
# Post a text message
swrite post "swrite is working!" -c "#general"

# Upload a file
echo "test content" > /tmp/test.txt
swrite upload -f /tmp/test.txt -c "#general" --comment "test upload"
```

---

## Token Summary

| Token | Where to find it | Used for |
|---|---|---|
| `xoxb-...` (Bot Token) | OAuth & Permissions → Bot User OAuth Token | All commands |

> swrite does **not** use App-Level Tokens (`xapp-`). Socket Mode is not required.

---

## Server Mode (Docker / Kubernetes)

For automated deployments, skip the config file and use environment variables instead:

```bash
export SWRITE_MODE=server
export SWRITE_TOKEN=xoxb-...
export SWRITE_CHANNEL="#alerts"

echo "container started" | swrite post
```

See the [README](../README.md#server-mode) for details.
