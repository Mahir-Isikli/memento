package capture

// #cgo LDFLAGS: -framework Carbon -framework CoreFoundation -framework CoreGraphics
// #include "keylogger.h"
// #include "keylogger.c"
import "C"

import (
	"sync"
	"time"
)

type KeyState uint8

const (
	KeyStateInvalid KeyState = iota
	KeyStateDown
	KeyStateUp
)

type KeyEvent struct {
	Key       string
	Char      rune
	State     KeyState
	Modifiers []string
	Timestamp time.Time
}

type Keylogger struct {
	running  bool
	mu       sync.Mutex
	callback func(KeyEvent)
}

var globalKeylogger *Keylogger
var lastKeyTime int64
var lastKeyCode int

//export handleKeyEvent
func handleKeyEvent(keyCode C.int, ch C.int, stateCode C.int, ctrl C.bool, opt C.bool, shift C.bool, cmd C.bool) {
	if globalKeylogger == nil || globalKeylogger.callback == nil {
		return
	}

	// Deduplicate - ignore if same key within 50ms
	now := time.Now().UnixMilli()
	if int(keyCode) == lastKeyCode && now-lastKeyTime < 50 {
		return
	}
	lastKeyCode = int(keyCode)
	lastKeyTime = now

	keyName := keyCodeToName(int(keyCode))

	var state KeyState
	switch stateCode {
	case 0:
		state = KeyStateUp
	case 1:
		state = KeyStateDown
	default:
		state = KeyStateInvalid
	}

	var modifiers []string
	if ctrl {
		modifiers = append(modifiers, "ctrl")
	}
	if opt {
		modifiers = append(modifiers, "alt")
	}
	if shift {
		modifiers = append(modifiers, "shift")
	}
	if cmd {
		modifiers = append(modifiers, "cmd")
	}

	event := KeyEvent{
		Key:       keyName,
		Char:      rune(ch),
		State:     state,
		Modifiers: modifiers,
		Timestamp: time.Now(),
	}

	globalKeylogger.callback(event)
}

func NewKeylogger(callback func(KeyEvent)) *Keylogger {
	kl := &Keylogger{
		callback: callback,
	}
	globalKeylogger = kl
	return kl
}

func (kl *Keylogger) Start() error {
	kl.mu.Lock()
	if kl.running {
		kl.mu.Unlock()
		return nil
	}
	kl.running = true
	kl.mu.Unlock()

	go func() {
		C.startKeylogger()
	}()

	return nil
}

func (kl *Keylogger) Stop() {
	kl.mu.Lock()
	kl.running = false
	kl.mu.Unlock()
	C.stopKeylogger()
}

func (kl *Keylogger) IsRunning() bool {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	return kl.running
}

var keyCodeNames = map[int]string{
	0: "a", 1: "s", 2: "d", 3: "f", 4: "h", 5: "g", 6: "z", 7: "x",
	8: "c", 9: "v", 11: "b", 12: "q", 13: "w", 14: "e", 15: "r",
	16: "y", 17: "t", 18: "1", 19: "2", 20: "3", 21: "4", 22: "6",
	23: "5", 24: "=", 25: "9", 26: "7", 27: "-", 28: "8", 29: "0",
	30: "]", 31: "o", 32: "u", 33: "[", 34: "i", 35: "p", 36: "return",
	37: "l", 38: "j", 39: "'", 40: "k", 41: ";", 42: "\\", 43: ",",
	44: "/", 45: "n", 46: "m", 47: ".", 48: "tab", 49: "space", 50: "`",
	51: "backspace", 53: "escape", 54: "rightcmd", 55: "leftcmd",
	56: "leftshift", 57: "capslock", 58: "leftoption", 59: "leftctrl",
	60: "rightshift", 61: "rightoption", 62: "rightctrl", 63: "fn",
	96: "f5", 97: "f6", 98: "f7", 99: "f3", 100: "f8", 101: "f9",
	103: "f11", 109: "f10", 111: "f12", 115: "home", 116: "pageup",
	117: "delete", 118: "f4", 119: "end", 120: "f2", 121: "pagedown",
	122: "f1", 123: "left", 124: "right", 125: "down", 126: "up",
}

func keyCodeToName(keyCode int) string {
	if name, ok := keyCodeNames[keyCode]; ok {
		return name
	}
	return "unknown"
}
