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
