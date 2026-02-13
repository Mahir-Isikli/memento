# Memento

Local-first screen recording & keystroke logging tool for macOS. Screenshots every 10 min, OCR search, all data stays in `~/.memento`.

## Project Structure

```
memento/
├── cmd/              # Go CLI entry point
├── internal/         # Core Go packages
├── scripts/          # Install/uninstall scripts
├── site/             # Landing page (deployed to Cloudflare Pages)
│   ├── index.html    # Main page with lightbox, lazy loading
│   ├── screenshots/  # WebP images for the marquee
│   └── llms.txt      # LLM-readable project info
├── SKILL.md          # Agent skill file for AI assistants
└── Makefile          # Build commands
```

## Website & Hosting

- **Domain:** `memento.cx` (also `www.memento.cx`)
- **Registrar:** Namecheap
- **DNS & Hosting:** Cloudflare Pages
- **Project name:** `memento-site`

### Deploy site changes:
```bash
npx wrangler pages deploy site --project-name memento-site
```

### Cloudflare API (stored in keychain):
```bash
# Get token
CF_TOKEN=$(security find-generic-password -s "cloudflare-api-token" -w)
ACCOUNT_ID=$(security find-generic-password -s "cloudflare-account-id" -w)

# List custom domains
curl -s "https://api.cloudflare.com/client/v4/accounts/${ACCOUNT_ID}/pages/projects/memento-site/domains" \
  -H "Authorization: Bearer ${CF_TOKEN}"

# Add custom domain
curl -X POST "https://api.cloudflare.com/client/v4/accounts/${ACCOUNT_ID}/pages/projects/memento-site/domains" \
  -H "Authorization: Bearer ${CF_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"name": "subdomain.memento.cx"}'
```

### Namecheap API (stored in keychain):
```bash
# Get credentials
NC_KEY=$(security find-generic-password -s "namecheap-api-key" -w)
NC_USER=$(security find-generic-password -s "namecheap-username" -w)
MY_IP=$(curl -s ifconfig.me)

# Check DNS settings
curl -s "https://api.namecheap.com/xml.response?ApiUser=${NC_USER}&ApiKey=${NC_KEY}&UserName=${NC_USER}&Command=namecheap.domains.dns.getList&ClientIp=${MY_IP}&SLD=memento&TLD=cx"
```

## CLI Commands

```bash
memento status                           # Check if running + stats
memento timeline --from "2 hours ago"    # Browse activity
memento search "query"                   # Search OCR + keystrokes
memento search "error" --from "today"    # Filter by time
memento search "slack" -o json           # JSON output for scripts
memento pause                            # Pause recording
memento uninstall                        # Clean removal
```

## Build

```bash
make build      # Build binary
make install    # Install to /usr/local/bin
```

## Site Features

- Marquee with real screenshots (WebP, lazy loaded)
- Lightbox: click image to expand, ESC or click to close
- Copy-to-clipboard for install command
- FAQ accordion
- Mobile responsive
