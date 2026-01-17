package capture

import (
	"strings"
	"sync"
	"time"
)

const (
	SessionIdleTimeout = 30 * time.Second
)

type TypingSessionBuffer struct {
	mu            sync.Mutex
	startTime     time.Time
	lastKeyTime   time.Time
	keys          []string
	keyCount      int
	app           string
	window        string
	onFlush       func(session *TypingSessionData)
}

type TypingSessionData struct {
	StartTime time.Time
	EndTime   time.Time
	Text      string
	KeyCount  int
	App       string
	Window    string
}

func NewTypingSessionBuffer(onFlush func(*TypingSessionData)) *TypingSessionBuffer {
	return &TypingSessionBuffer{
		onFlush: onFlush,
	}
}

func (b *TypingSessionBuffer) AddKey(key string, app, window string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	// Check if we need to start a new session
	shouldStartNew := false
	if b.startTime.IsZero() {
		shouldStartNew = true
	} else if app != b.app || window != b.window {
		// Context changed - flush and start new
		b.flushLocked()
		shouldStartNew = true
	} else if now.Sub(b.lastKeyTime) > SessionIdleTimeout {
		// Idle timeout - flush and start new
		b.flushLocked()
		shouldStartNew = true
	}

	if shouldStartNew {
		b.startTime = now
		b.keys = nil
		b.keyCount = 0
		b.app = app
		b.window = window
	}

	// Add the key
	b.keys = append(b.keys, keyToText(key))
	b.keyCount++
	b.lastKeyTime = now
}

func (b *TypingSessionBuffer) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flushLocked()
}

func (b *TypingSessionBuffer) flushLocked() {
	if b.keyCount == 0 || b.startTime.IsZero() {
		return
	}

	text := buildText(b.keys)
	
	session := &TypingSessionData{
		StartTime: b.startTime,
		EndTime:   b.lastKeyTime,
		Text:      text,
		KeyCount:  b.keyCount,
		App:       b.app,
		Window:    b.window,
	}

	// Reset buffer
	b.startTime = time.Time{}
	b.keys = nil
	b.keyCount = 0

	// Call flush callback (outside lock would be better, but keep simple for now)
	if b.onFlush != nil {
		b.onFlush(session)
	}
}

func keyToText(key string) string {
	switch key {
	case "space":
		return " "
	case "return":
		return "\n"
	case "tab":
		return "\t"
	case "backspace":
		return "\b"
	case "delete", "unknown":
		return ""
	case "escape", "leftshift", "rightshift", "leftcmd", "rightcmd",
		"leftoption", "rightoption", "leftctrl", "rightctrl",
		"capslock", "fn", "up", "down", "left", "right",
		"home", "end", "pageup", "pagedown",
		"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12":
		return ""
	default:
		return key
	}
}

func buildText(keys []string) string {
	var result strings.Builder
	for _, k := range keys {
		if k == "\b" {
			// Handle backspace - remove last character if possible
			s := result.String()
			if len(s) > 0 {
				result.Reset()
				result.WriteString(s[:len(s)-1])
			}
		} else {
			result.WriteString(k)
		}
	}
	return strings.TrimSpace(result.String())
}

// StartIdleChecker starts a goroutine that periodically checks for idle sessions and flushes them
func (b *TypingSessionBuffer) StartIdleChecker(interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.mu.Lock()
				if b.keyCount > 0 && !b.lastKeyTime.IsZero() {
					if time.Since(b.lastKeyTime) > SessionIdleTimeout {
						b.flushLocked()
					}
				}
				b.mu.Unlock()
			case <-stop:
				return
			}
		}
	}()
	return stop
}
