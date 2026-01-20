---
name: memento
description: Search your computer history - screenshots, keystrokes, OCR text. Use when the user asks about something they saw, typed, or did on their computer. Helps find forgotten information, recall context, or search through past activity.
---

# Memento - Your Digital Memory

Search everything you've seen and typed on your Mac.

## Quick start

```bash
memento search "that error message"     # Search everything
memento search "api key" --type keys    # Search keystrokes only
memento search "slack" --type ocr       # Search OCR text only
memento timeline                        # Browse recent activity
memento keys --today                    # What you typed today
memento status                          # Check if running + stats
```

## When to use

- User asks "what was that thing I saw earlier?"
- User needs to find something they typed but forgot where
- User wants to recall context from a past session
- User is looking for a URL, error message, or snippet they saw
- User asks "when did I last work on X?"

## Search commands

```bash
# Full-text search across screenshots (OCR) and keystrokes
memento search "query"

# Filter by type
memento search "query" --type ocr       # Screenshots only
memento search "query" --type keys      # Keystrokes only

# Filter by time
memento search "query" --after "2024-01-15"
memento search "query" --before "2024-01-20"
memento search "query" --today
memento search "query" --yesterday
memento search "query" --week           # Last 7 days

# Filter by app
memento search "query" --app "VS Code"
memento search "query" --app Terminal

# Combine filters
memento search "error" --type ocr --app Terminal --today

# Limit results
memento search "query" --limit 5
```

## Browse activity

```bash
memento timeline                  # Interactive timeline browser
memento timeline --today          # Today's activity
memento timeline --date 2024-01-15

memento keys --today              # Keystrokes from today
memento keys --yesterday
memento keys --app Slack          # What you typed in Slack

memento screenshots --today       # List today's screenshots
memento screenshots --open 42     # Open screenshot #42
```

## JSON output (for agents)

All commands support `-o json` for structured output:

```bash
memento search "api" -o json
memento keys --today -o json
memento status -o json
```

Example JSON response:
```json
{
  "results": [
    {
      "type": "ocr",
      "timestamp": "2024-01-15T14:32:00Z",
      "app": "Terminal",
      "window": "zsh",
      "text": "export API_KEY=sk-...",
      "screenshot_id": 1234
    }
  ],
  "total": 1,
  "query": "api"
}
```

## Status & control

```bash
memento status                    # Running? Stats? Last capture?
memento pause                     # Temporarily stop capturing
memento resume                    # Resume capturing
memento capture                   # Force immediate capture
```

## Configuration

```bash
memento config                    # Show current config
memento config set interval 300   # Capture every 5 min (default: 600)
memento config set quality 70     # WebP quality (default: 80)
memento config add-exclude Safari # Don't capture Safari
memento config remove-exclude Safari
```

## Example agent workflows

### Find something the user saw
```bash
# User: "what was that error I got in the terminal?"
memento search "error" --app Terminal --today -o json --limit 5
```

### Recall a URL or link
```bash
# User: "what was that article I was reading?"
memento search "http" --type ocr --today -o json
```

### Find something typed
```bash
# User: "what was that command I ran earlier?"
memento keys --app Terminal --today -o json
```

### Get context for a time period
```bash
# User: "what was I working on yesterday afternoon?"
memento timeline --yesterday -o json
```

## Data location

All data stored locally in `~/.memento/`:
- `screenshots/` - WebP images (~100KB each)
- `memento.db` - SQLite database (keystrokes, OCR, metadata)
- `config.json` - User configuration

## Privacy notes

- All data stays local by default
- Passwords are not captured (excluded input types)
- Add sensitive apps to exclusion list
- Optional encrypted backup to Cloudflare R2
