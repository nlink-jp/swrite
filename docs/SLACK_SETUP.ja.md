# swrite Slack セットアップガイド

`swrite` を使用するには、Slack アプリを作成して **ボットトークン**（`xoxb-`）を取得する必要があります。
stail とは異なり、swrite は App-Level Token や Socket Mode を必要としません。

---

## Step 1: Slack アプリを作成する

1. [Slack API サイト](https://api.slack.com/apps) にアクセスしてログインします。

2. **"Create New App"** をクリックします。

3. **"From scratch"** を選択します。

4. アプリ名（例: `swrite`）を入力し、ワークスペースを選択して **"Create App"** をクリックします。

### 表示情報（任意）

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

## Step 2: ボットトークンのスコープを追加する

1. 左サイドバーの **"OAuth & Permissions"** をクリックします。

2. **"Scopes"** セクションまでスクロールします。

3. **"Bot Token Scopes"** の下に以下のスコープを追加します。

   **必須:**

   | スコープ | 用途 |
   |---|---|
   | `chat:write` | メッセージを投稿する（`swrite post`） |
   | `files:write` | ファイルをアップロードする（`swrite upload`） |
   | `channels:read` | パブリックチャンネル名を ID に解決する |

   **推奨:**

   | スコープ | 用途 |
   |---|---|
   | `groups:read` | プライベートチャンネル名を ID に解決する |
   | `channels:join` | `not_in_channel` 時に自動参加（手動 `/invite` が不要になる） |
   | `im:write` | ダイレクトメッセージを送信する（`--user` フラグ） |

> **注意:** `channels:join` を省略した場合は、投稿先のパブリックチャンネルそれぞれで
> `/invite @<アプリ名>` を実行してボットを手動招待する必要があります。

---

## Step 3: アプリをワークスペースにインストールする

1. 左サイドバーの **"OAuth & Permissions"** をクリックします。

2. ページ上部の **"Install to Workspace"** をクリックします。

3. **"Allow"** をクリックして承認します。

---

## Step 4: ボットトークンをコピーする

インストール後、**"OAuth & Permissions"** ページに **"Bot User OAuth Token"**（`xoxb-` で始まる）が表示されます。次のステップで使用するためコピーしておきます。

---

## Step 5: swrite を設定する

### 設定ファイルを初期化する

```bash
swrite config init
```

`~/.config/swrite/config.json` がパーミッション `0600` で作成されます。

### プロファイルを追加する

```bash
swrite profile add my-workspace --channel "#alerts"
```

ボットトークンの入力を求められます：

```
Bot Token (xoxb-...): [xoxb- トークンを貼り付け]
```

### アクティブなプロファイルを設定する

```bash
swrite profile use my-workspace
```

セットアップ完了です。

---

## Step 6: プライベートチャンネルにボットを招待する

**プライベートチャンネル**にはボットを手動で招待する必要があります。
投稿したいプライベートチャンネルで次のコマンドを実行します：

```
/invite @<アプリ名>
```

**パブリックチャンネル**については、`channels:join` スコープを付与していれば初回利用時に自動参加します。

---

## 動作確認

正しく設定できているかテストします：

```bash
# テキストメッセージを投稿
swrite post "swrite 動作確認" -c "#general"

# ファイルをアップロード
echo "テストコンテンツ" > /tmp/test.txt
swrite upload -f /tmp/test.txt -c "#general" --comment "テストアップロード"
```

---

## トークン一覧

| トークン | 取得場所 | 用途 |
|---|---|---|
| `xoxb-...`（ボットトークン） | OAuth & Permissions → Bot User OAuth Token | 全コマンド |

> swrite は App-Level Token（`xapp-`）を使用しません。Socket Mode も不要です。

---

## サーバーモード（Docker / Kubernetes）

自動化された環境では、設定ファイルを使わず環境変数から設定を読み込めます：

```bash
export SWRITE_MODE=server
export SWRITE_TOKEN=xoxb-...
export SWRITE_CHANNEL="#alerts"

echo "コンテナ起動" | swrite post
```

詳細は [README](../README.md#server-mode) を参照してください。
