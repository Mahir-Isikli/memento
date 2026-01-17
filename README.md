# Memento

Personal digital memory archive for macOS. Captures screenshots every 10 minutes, logs keystrokes with active window context, runs OCR on captured images, and provides an agent-friendly CLI for querying your digital history.

Inspired by [Tobi Lutke's](https://www.acquired.fm/episodes/shopify-tobi-lutke) 15-year personal archiving setup.

## Features

- **Screenshot capture** every 10 minutes (WebP format, ~150KB each)
- **Keystroke logging** with active window/app context
- **OCR processing** using macOS Vision framework
- **Agent-friendly CLI** with `--json` and `--plain` output modes
- **Daily backups** to Cloudflare R2 via rclone
- **LaunchAgent** for auto-start on login

## Installation

### Build from source

```bash
git clone https://github.com/mahirisikli/memento.git
cd memento
make build
make install  # installs to /usr/local/bin
```

### Dependencies

- Go 1.21+
- macOS 12+ (for Vision framework OCR)
- uv (Python package manager): `brew install uv`
- rclone for R2 backup: `brew install rclone`

### OCR Setup

OCR uses `ocrmac` in a dedicated venv at `~/.memento/.venv`:

```bash
make deps  # Sets up venv and installs ocrmac
```

Or manually:
```bash
mkdir -p ~/.memento
uv venv ~/.memento/.venv
~/.memento/.venv/bin/pip install ocrmac
```

## Permissions

Memento requires these macOS permissions:

1. **Screen Recording** (for screenshots)
   - System Settings > Privacy & Security > Screen Recording
   - Add Terminal or memento binary

2. **Accessibility** (for keystroke logging)
   - System Settings > Privacy & Security > Accessibility
   - Add Terminal or memento binary

## Usage

### Quick start

```bash
# Take a single screenshot
memento capture

# Start the daemon (foreground)
memento start

# Check status
memento status
```

### Search

```bash
# Search OCR text and window titles
memento search "meeting notes"
memento search "docker" --from "2 days ago"
memento search "react" --json  # for agents
```

### Timeline

```bash
# View today's activity
memento timeline

# View specific date
memento timeline --date 2026-01-15
```

### Keystroke history

```bash
# Show today's keystroke summary
memento keys --today

# Filter by app
memento keys --from "2 hours ago" --app "VS Code"
```

### Screenshots

```bash
# List today's screenshots
memento screenshots list --today

# Open a screenshot
memento screenshots show 42

# View OCR text
memento screenshots ocr 42
```

### Configuration

```bash
# Show config
memento config show

# Set screenshot interval (seconds)
memento config set interval 600

# Set WebP quality (1-100)
memento config set quality 80

# Enable backup
memento config set backup_enabled true
memento config set r2_bucket my-backup-bucket
```

### Backup

```bash
# Check backup status
memento backup status

# Trigger immediate backup
memento backup now
```

## Auto-start with LaunchAgent

```bash
# Install LaunchAgent
make launchagent

# Start
launchctl load ~/Library/LaunchAgents/com.memento.daemon.plist

# Stop
launchctl unload ~/Library/LaunchAgents/com.memento.daemon.plist
```

## Output Formats

All commands support three output formats:

- **text** (default): Human-readable
- **json**: Machine-readable for agents (`-o json` or `MEMENTO_JSON=1`)
- **plain**: Tab-separated for scripting (`-o plain` or `MEMENTO_PLAIN=1`)

## Storage

Data is stored in `~/.memento/`:

```
~/.memento/
├── config.json
├── memento.db          # SQLite database
├── screenshots/
│   └── 2026/01/17/     # Organized by date
│       └── 2026-01-17_10-00-00.webp
└── logs/
    └── memento.log
```

## Storage Estimates

- ~144 screenshots/day × ~150KB = ~21MB/day
- ~630MB/month
- ~7.5GB/year
- 15 years = ~112GB

## License

MIT
