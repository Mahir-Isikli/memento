package capture

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ScreenshotCapture struct {
	quality    int
	fullScreen bool
	tempDir    string
}

func NewScreenshotCapture(quality int, fullScreen bool) *ScreenshotCapture {
	if quality <= 0 || quality > 100 {
		quality = 80
	}
	return &ScreenshotCapture{
		quality:    quality,
		fullScreen: fullScreen,
		tempDir:    os.TempDir(),
	}
}

type CaptureResult struct {
	Timestamp time.Time
	Data      []byte
	Width     int
	Height    int
}

func (sc *ScreenshotCapture) Capture() (*CaptureResult, error) {
	timestamp := time.Now()

	tempPNG := fmt.Sprintf("%s/memento_temp_%d.png", sc.tempDir, timestamp.UnixNano())
	tempWebP := fmt.Sprintf("%s/memento_temp_%d.webp", sc.tempDir, timestamp.UnixNano())
	defer os.Remove(tempPNG)
	defer os.Remove(tempWebP)

	// Capture screenshot using macOS screencapture CLI
	// -x: no sound, -C: include cursor
	cmd := exec.Command("screencapture", "-x", "-C", tempPNG)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := string(output)
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("screencapture failed (check Screen Recording permissions): %s", errMsg)
	}

	// Get original image dimensions using sips (built-in macOS tool)
	origWidth, origHeight, err := getImageDimensions(tempPNG)
	if err != nil {
		return nil, fmt.Errorf("failed to get image dimensions: %w", err)
	}

	// Resize to half resolution to save storage (~68% reduction)
	width := origWidth / 2
	height := origHeight / 2
	cmd = exec.Command("sips", "-z", fmt.Sprintf("%d", height), fmt.Sprintf("%d", width), tempPNG)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to resize image: %w", err)
	}

	// Convert to WebP using cwebp CLI (much more memory efficient than Go library)
	cmd = exec.Command("cwebp", "-q", fmt.Sprintf("%d", sc.quality), "-quiet", tempPNG, "-o", tempWebP)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cwebp failed (install with: brew install webp): %w", err)
	}

	// Read the WebP file
	webpData, err := os.ReadFile(tempWebP)
	if err != nil {
		return nil, fmt.Errorf("failed to read WebP file: %w", err)
	}

	return &CaptureResult{
		Timestamp: timestamp,
		Data:      webpData,
		Width:     width,
		Height:    height,
	}, nil
}

func (sc *ScreenshotCapture) CaptureToFile(filepath string) (*CaptureResult, error) {
	result, err := sc.Capture()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath, result.Data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write screenshot: %w", err)
	}

	return result, nil
}

func getImageDimensions(filepath string) (int, int, error) {
	// Use sips (built-in macOS) to get dimensions without loading image into memory
	cmd := exec.Command("sips", "-g", "pixelWidth", "-g", "pixelHeight", filepath)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(output), "\n")
	var width, height int
	for _, line := range lines {
		if strings.Contains(line, "pixelWidth") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				width, _ = strconv.Atoi(parts[len(parts)-1])
			}
		}
		if strings.Contains(line, "pixelHeight") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				height, _ = strconv.Atoi(parts[len(parts)-1])
			}
		}
	}
	return width, height, nil
}

func GetActiveWindowID() (string, error) {
	script := `
		tell application "System Events"
			set frontApp to first application process whose frontmost is true
			set frontWindow to first window of frontApp
			return id of frontWindow
		end tell
	`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
}
