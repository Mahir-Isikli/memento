package capture

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"time"

	"github.com/chai2010/webp"
)

type ScreenshotCapture struct {
	quality     int
	fullScreen  bool
	tempDir     string
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
	
	tempFile := fmt.Sprintf("%s/memento_temp_%d.png", sc.tempDir, timestamp.UnixNano())
	defer os.Remove(tempFile)
	
	// -x: no sound, -C: include cursor
	// Always capture full screen - active window capture via -l is unreliable
	args := []string{"-x", "-C", tempFile}
	
	cmd := exec.Command("screencapture", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := string(output)
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("screencapture failed (check Screen Recording permissions in System Settings > Privacy & Security): %s", errMsg)
	}
	
	pngData, err := os.ReadFile(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshot: %w", err)
	}
	
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}
	
	var webpBuf bytes.Buffer
	options := &webp.Options{
		Lossless: false,
		Quality:  float32(sc.quality),
	}
	if err := webp.Encode(&webpBuf, img, options); err != nil {
		return nil, fmt.Errorf("failed to encode WebP: %w", err)
	}
	
	bounds := img.Bounds()
	return &CaptureResult{
		Timestamp: timestamp,
		Data:      webpBuf.Bytes(),
		Width:     bounds.Dx(),
		Height:    bounds.Dy(),
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

func DecodeWebP(data []byte) (image.Image, error) {
	return webp.Decode(bytes.NewReader(data))
}
