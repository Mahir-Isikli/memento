#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                    MEMENTO INSTALLER                      ║"
echo "║         Personal Digital Memory Archive for macOS         ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check macOS
if [[ "$(uname)" != "Darwin" ]]; then
    echo -e "${RED}Error: Memento only works on macOS${NC}"
    exit 1
fi

# Check for required tools
echo "Checking dependencies..."

if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Go not found. Installing via Homebrew...${NC}"
    if ! command -v brew &> /dev/null; then
        echo -e "${RED}Error: Homebrew required. Install from https://brew.sh${NC}"
        exit 1
    fi
    brew install go
fi

if ! command -v cwebp &> /dev/null; then
    echo -e "${YELLOW}WebP tools not found. Installing via Homebrew...${NC}"
    brew install webp
fi

if ! command -v uv &> /dev/null; then
    echo -e "${YELLOW}uv not found. Installing via Homebrew...${NC}"
    brew install uv
fi

echo -e "${GREEN}✓ All dependencies installed${NC}"

# Clone or update repo
INSTALL_DIR="${HOME}/.memento-src"
if [ -d "$INSTALL_DIR" ]; then
    echo "Updating existing installation..."
    cd "$INSTALL_DIR"
    git pull
else
    echo "Downloading memento..."
    git clone https://github.com/Mahir-Isikli/memento.git "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# Build
echo "Building memento..."
go build -o memento ./cmd/memento

# Install binary to ~/.local/bin (no sudo needed)
INSTALL_BIN="${HOME}/.local/bin"
mkdir -p "$INSTALL_BIN"
echo "Installing to $INSTALL_BIN..."
cp memento "$INSTALL_BIN/memento"
chmod +x "$INSTALL_BIN/memento"

# Ensure ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$INSTALL_BIN:"* ]]; then
    echo -e "${YELLOW}Adding ~/.local/bin to PATH...${NC}"
    SHELL_RC=""
    if [ -f ~/.zshrc ]; then
        SHELL_RC=~/.zshrc
    elif [ -f ~/.bashrc ]; then
        SHELL_RC=~/.bashrc
    elif [ -f ~/.bash_profile ]; then
        SHELL_RC=~/.bash_profile
    fi
    if [ -n "$SHELL_RC" ]; then
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$SHELL_RC"
        echo "Added to $SHELL_RC - restart terminal or run: source $SHELL_RC"
    fi
fi

# Create data directory
mkdir -p ~/.memento

# Setup OCR
echo "Setting up OCR (this may take a minute)..."
if [ ! -d ~/.memento/.venv ]; then
    uv venv ~/.memento/.venv
fi
~/.memento/.venv/bin/pip install -q ocrmac

# Create LaunchAgent
echo "Setting up auto-start..."
mkdir -p ~/Library/LaunchAgents
cat > ~/Library/LaunchAgents/com.memento.daemon.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.memento.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_BIN}/memento</string>
        <string>start</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>/tmp/memento.stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/memento.stderr.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>${HOME}/.local/bin:/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>
EOF

# Start memento (this will trigger permission popups)
echo ""
echo -e "${GREEN}Starting memento...${NC}"
echo "Permission dialogs will appear - please grant access!"
echo ""

# Run memento once to trigger permission dialogs
"$INSTALL_BIN/memento" start &
MEMENTO_PID=$!
sleep 3
kill $MEMENTO_PID 2>/dev/null || true

# Now load the LaunchAgent for auto-start
launchctl load ~/Library/LaunchAgents/com.memento.daemon.plist 2>/dev/null || true

echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗"
echo -e "║              INSTALLATION COMPLETE! 🎉                     ║"
echo -e "╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}If you saw permission dialogs, click 'Open System Settings' and enable memento.${NC}"
echo -e "${YELLOW}Then restart memento with:${NC}"
echo "  launchctl unload ~/Library/LaunchAgents/com.memento.daemon.plist"
echo "  launchctl load ~/Library/LaunchAgents/com.memento.daemon.plist"
echo ""
echo -e "${GREEN}Commands:${NC}"
echo "  memento status         # Check stats"
echo "  memento search \"text\"  # Search your history"
echo "  memento keys --today   # See what you typed today"
echo "  memento timeline       # Browse activity"
echo ""
echo -e "${GREEN}Optional - Enable R2 backup:${NC}"
echo "  brew install rclone"
echo "  rclone config          # Setup R2 remote named 'r2'"
echo "  memento config set backup_enabled true"
echo "  memento config set r2_bucket YOUR_BUCKET_NAME"
echo ""
