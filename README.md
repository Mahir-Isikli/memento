# Memento

> Your personal digital memory. Screenshots, keystrokes, OCR — all local, all searchable.

```bash
curl -fsSL https://raw.githubusercontent.com/Mahir-Isikli/memento/master/scripts/install-memento.sh | bash
```

## What it does

Every 10 minutes, memento captures what's on your screen and what you're typing. Everything stays on your machine and is fully searchable.

| Captured | Details |
|----------|---------|
| Screenshots | Half-resolution WebP (~100KB each) |
| Keystrokes | Aggregated into typing sessions |
| Window context | App name + window title |
| OCR text | Extracted from screenshots |

**Not captured:** passwords, background windows, audio, mouse clicks.

## Usage

```bash
memento search "that error I saw"    # Search everything
memento keys --today                  # What you typed today  
memento timeline                      # Browse activity
memento status                        # Stats
```

All commands support `-o json` for scripts/agents.

## Local Wispr Flow history

Import the transcript history already stored by Wispr Flow on this Mac:

```bash
memento wispr sync
memento dictations "launch" --from today
memento search "launch" -o json
```

The import reads `~/Library/Application Support/Wispr Flow/flow.sqlite` without changing it.
It preserves the raw transcript, formatted text, edits, pasted text, app, URL and timestamp.
It copies no audio or screenshot blobs and sends nothing to a server.
Repeated imports update existing entries rather than creating duplicates.
Deleting a transcript in Wispr does not delete the archived copy in Memento.

To import new transcripts every 5 minutes while capturing activity:

```bash
memento start --keys=true --wispr=true
```

Search now covers screenshot text, typing sessions and imported dictations.
Typing uses keyboard-layout characters and skips Command and Control shortcuts.
The capture callback skips macOS secure keyboard entry.
Dictation and pasted text need a source integration because key events do not contain inserted text.

## Semantic search

```bash
memento semantic setup                     # Download the public model once
memento semantic index                     # Index new or changed records
memento search "preparing a launch announcement" --semantic --limit 5
memento semantic status -o json
```

Semantic search uses a quantised 24MB MiniLM embedding model with one CPU thread.
It combines meaning-based similarity with word matches across screen text, typing and Wispr transcripts.
Vectors stay in `~/.memento/semantic.sqlite`. The pinned model files stay in `~/.memento/models/`.
The helper runs only during an index or search command, then exits. There is no model server.
Search indexes new or changed records automatically. An unchanged index does not load the model.
Long records are split into overlapping chunks, and results preserve the original text.
The model is best suited to English. Exact word search remains available for names and other languages.
Only `semantic setup` downloads files from Hugging Face. Indexing and searching use local files and no network calls.

## Storage

~6 MB/day → ~180 MB/month → **4+ years in 10GB**

## Configuration

```bash
memento config set interval 300      # Every 5 min instead of 10
memento config set quality 70        # Smaller files
```

## Cloud Backup (Optional)

```bash
brew install rclone
rclone config                        # Setup Cloudflare R2
memento config set backup_enabled true
memento config set r2_bucket my-bucket
```

Backups run daily at 2-4 AM. R2 has 10GB free tier.

## Uninstall

```bash
launchctl unload ~/Library/LaunchAgents/com.memento.daemon.plist
rm ~/Library/LaunchAgents/com.memento.daemon.plist ~/.local/bin/memento
rm -rf ~/.memento ~/.memento-src
```

## Requirements

- macOS 12+
- Homebrew

## How it works

- Screenshots via `screencapture` → resized with `sips` → compressed with `cwebp`
- Keystrokes via CGEventTap (Accessibility permission)
- OCR via macOS Vision framework (`ocrmac`)
- Storage in SQLite at `~/.memento/`
- Runs as LaunchAgent (auto-starts on login)

## Privacy

All data stays local. Optional R2 backup goes to your own Cloudflare account.

## License

MIT
