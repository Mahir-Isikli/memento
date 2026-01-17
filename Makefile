.PHONY: build install clean test run

BINARY=memento
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/memento

install: build
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)
	sudo chmod +x /usr/local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
	go clean

test:
	go test ./...

run: build
	./$(BINARY) start

capture: build
	./$(BINARY) capture

status: build
	./$(BINARY) status

# Install dependencies
deps:
	go mod tidy
	@echo "Setting up OCR venv in ~/.memento/.venv..."
	mkdir -p ~/.memento
	uv venv ~/.memento/.venv
	~/.memento/.venv/bin/pip install ocrmac

# Setup LaunchAgent
launchagent:
	mkdir -p ~/Library/LaunchAgents
	cp scripts/launchagent.plist ~/Library/LaunchAgents/com.memento.daemon.plist
	@echo "To start: launchctl load ~/Library/LaunchAgents/com.memento.daemon.plist"

# Uninstall LaunchAgent
uninstall-launchagent:
	launchctl unload ~/Library/LaunchAgents/com.memento.daemon.plist 2>/dev/null || true
	rm -f ~/Library/LaunchAgents/com.memento.daemon.plist
