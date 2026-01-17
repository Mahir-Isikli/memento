#!/bin/bash
set -e

echo "Installing Memento..."

# Build the binary
echo "Building..."
cd "$(dirname "$0")/.."
go build -o memento ./cmd/memento

# Install to /usr/local/bin
echo "Installing binary to /usr/local/bin..."
sudo cp memento /usr/local/bin/memento
sudo chmod +x /usr/local/bin/memento

# Create LaunchAgent
echo "Setting up LaunchAgent..."
mkdir -p ~/Library/LaunchAgents
cp scripts/launchagent.plist ~/Library/LaunchAgents/com.memento.daemon.plist

# Remind about permissions
echo ""
echo "Installation complete!"
echo ""
echo "IMPORTANT: Grant Accessibility permissions to memento for keystroke logging:"
echo "  System Settings > Privacy & Security > Accessibility"
echo "  Add: /usr/local/bin/memento"
echo ""
echo "To start memento:"
echo "  launchctl load ~/Library/LaunchAgents/com.memento.daemon.plist"
echo ""
echo "To stop memento:"
echo "  launchctl unload ~/Library/LaunchAgents/com.memento.daemon.plist"
echo ""
echo "To run manually (foreground):"
echo "  memento start"
