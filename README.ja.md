# swrite

コマンドラインから Slack にメッセージとファイルを投稿するツール。

ボットワークフローとシェルパイプライン向けに設計されています。[stail](https://github.com/nlink-jp/stail) および [slack-router](https://github.com/nlink-jp/slack-router) と組み合わせて使用することを想定しています。

## 特徴

- テキストまたは Block Kit メッセージをチャンネルや DM に投稿
- stdin を行ごとにストリーム、3秒ごとにバッチ送信（`--stream`）
- ファイルをコメント付きでアップロード
- プロファイルベースの設定でワークスペースを簡単に切り替え
- サーバーモード — 環境変数だけで設定ファイル不要（Docker、Kubernetes 向け）
- `not_in_channel` エラー時に自動チャンネル参加してリトライ

## セットアップ

初めて使う方は **[Slack セットアップガイド](docs/SLACK_SETUP.ja.md)** を参照してください。Slack アプリの作成とボットトークンの取得手順を説明しています。

## インストール

[Releases](https://github.com/nlink-jp/swrite/releases) から最新バイナリをダウンロード、またはソースからビルド：

```bash
make build   # dist/swrite を生成
```

## クイックスタート

```bash
# 1. 設定ファイルを作成
swrite config init

# 2. プロファイルを追加（ボットトークンをプロンプトで入力）
swrite profile add myworkspace --channel "#general"

# 3. メッセージを投稿
echo "デプロイ完了" | swrite post -c "#ops"
```

## コマンド

### `swrite post`

テキストまたは Block Kit メッセージを投稿します。

```
swrite post [メッセージ] [フラグ]

フラグ:
  -c, --channel string     送信先チャンネル（プロファイルのデフォルトを上書き）
      --user string        ユーザー ID へ DM として送信
  -f, --from-file string   ファイルからメッセージ本文を読み込む
  -s, --stream             stdin を行ごとにストリーム、3秒ごとにバッチ送信
  -t, --tee                投稿前に stdin を stdout にエコー
  -u, --username string    この投稿の表示名を上書き
  -i, --icon-emoji string  アイコン絵文字を上書き（例: :robot_face:）
      --format string      メッセージ形式: text または blocks（デフォルト: text）
```

**使用例:**

```bash
# 引数からテキスト投稿
swrite post "hello world" -c "#general"

# stdin から投稿
echo "サーバー起動完了" | swrite post -c "#ops"

# ファイルから Block Kit 投稿
cat payload.json | swrite post --format blocks -c "#alerts"

# ストリーム（tail -f や slack-router との組み合わせに有用）
tail -f /var/log/app.log | swrite post --stream -c "#logs"
```

### `swrite upload`

Slack にファイルをアップロードします。

```
swrite upload [フラグ]

フラグ:
  -c, --channel string    送信先チャンネル（プロファイルのデフォルトを上書き）
      --user string       ユーザー ID へ DM として送信
  -f, --file string       アップロードするファイルのパス、または stdin の場合は "-"（必須）
  -m, --comment string    ファイルに添付する初期コメント
  -n, --filename string   Slack 上に表示されるファイル名（デフォルト: --file のベース名）
```

**使用例:**

```bash
swrite upload -f report.csv -c "#data" --comment "週次レポート"
cat output.log | swrite upload -f - -c "#ops" --filename "run.log"
```

### `swrite config init`

`~/.config/swrite/config.json` にデフォルト設定ファイルを作成します。

### `swrite cache`

swrite はチャンネル一覧を `~/.config/swrite/cache/<profile>/` に 1 時間キャッシュし、
毎回の API 呼び出しを省略します。

```bash
swrite cache clear   # アクティブプロファイルのキャッシュを削除
```

### `swrite profile`

名前付きプロファイルを管理します。

```bash
swrite profile add myworkspace --channel "#general"   # トークンをプロンプトで入力
swrite profile list
swrite profile use myworkspace
swrite profile set channel "#ops"
swrite profile set token                              # セキュアなプロンプト
swrite profile remove old-workspace
```

## 設定

設定ファイルは `~/.config/swrite/config.json`（パーミッション 0600）に保存されます。
スキーマは [stail](https://github.com/nlink-jp/stail) と互換性があります。

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

### プロファイルフィールド

| フィールド | 必須 | 説明 |
|---|---|---|
| `provider` | はい | 常に `slack` |
| `token` | はい | Slack ボットトークン（`xoxb-...`） |
| `channel` | いいえ | デフォルト送信先チャンネル（`#名前` または ID） |
| `username` | いいえ | デフォルト表示名（上書き） |

## サーバーモード

`SWRITE_MODE=server` を設定すると、設定ファイルを完全にスキップし、環境変数から設定を読み込みます。Docker や Kubernetes でボットとして動作させる場合に使用します。

| 変数 | 必須 | 説明 |
|---|---|---|
| `SWRITE_MODE` | はい | `server` に設定 |
| `SWRITE_TOKEN` | はい | Slack ボットトークン |
| `SWRITE_CHANNEL` | いいえ | デフォルトチャンネル |
| `SWRITE_USERNAME` | いいえ | デフォルト表示名 |
| `SWRITE_CACHE_DIR` | いいえ | チャンネル一覧のキャッシュディレクトリ（繰り返し実行する場合に推奨） |

**Docker での例:**

```bash
docker run --rm \
  -e SWRITE_MODE=server \
  -e SWRITE_TOKEN=xoxb-... \
  -e SWRITE_CHANNEL="#alerts" \
  myimage sh -c 'echo "コンテナ起動" | swrite post'
```

## 必要な Slack スコープ

| スコープ | 用途 |
|---|---|
| `chat:write` | `post` |
| `files:write` | `upload` |
| `channels:read` | チャンネル名の解決 |
| `groups:read` | プライベートチャンネル名の解決 |
| `channels:join` | `not_in_channel` 時の自動参加 |
| `im:write` | `--user` による DM |

## グローバルフラグ

```
  --config string      設定ファイルのパス（デフォルト: ~/.config/swrite/config.json）
  -p, --profile string  使用するプロファイル（現在のプロファイルを上書き）
  -q, --quiet          標準エラー出力の情報メッセージを抑制
```
