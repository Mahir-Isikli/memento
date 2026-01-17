package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mahirisikli/memento/internal/capture"
	"github.com/mahirisikli/memento/internal/ocr"
	"github.com/mahirisikli/memento/internal/storage"
	"github.com/spf13/cobra"
)

var (
	screenshotInterval int
	screenshotQuality  int
	fullScreen         bool
	enableKeylogger    bool
	enableOCR          bool
)

func init() {
	startCmd.Flags().IntVar(&screenshotInterval, "interval", 600, "Screenshot interval in seconds")
	startCmd.Flags().IntVar(&screenshotQuality, "quality", 80, "WebP quality (1-100)")
	startCmd.Flags().BoolVar(&fullScreen, "fullscreen", false, "Capture full screen instead of active window")
	startCmd.Flags().BoolVar(&enableKeylogger, "keys", true, "Enable keystroke logging")
	startCmd.Flags().BoolVar(&enableOCR, "ocr", true, "Enable OCR processing")
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the memento daemon",
	Long:  `Start the memento daemon in the foreground. Captures screenshots at regular intervals and optionally logs keystrokes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemon()
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the memento daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("To stop memento, press Ctrl+C in the terminal where it's running, or kill the process.")
		fmt.Println("For LaunchAgent: launchctl unload ~/Library/LaunchAgents/com.memento.plist")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status and statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		stats, err := db.GetStats()
		if err != nil {
			return fmt.Errorf("failed to get stats: %w", err)
		}

		format := getOutputFormat()
		switch format {
		case "json":
			outputJSON(stats)
		case "plain":
			headers := []string{"key", "value"}
			var rows [][]string
			for k, v := range stats {
				rows = append(rows, []string{k, fmt.Sprintf("%v", v)})
			}
			outputPlain(headers, rows)
		default:
			fmt.Println("Memento Status")
			fmt.Println("==============")
			for k, v := range stats {
				fmt.Printf("%-20s: %v\n", k, v)
			}
		}
		return nil
	},
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func runDaemon() error {
	storagePath := getStoragePath()
	log.Printf("Starting memento daemon with storage at %s", storagePath)

	db, err := storage.NewDB(storagePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fm := storage.NewFileManager(storagePath)
	if err := fm.EnsureLogsDir(); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	screenshotCapture := capture.NewScreenshotCapture(screenshotQuality, fullScreen)
	ocrEngine := ocr.NewOCREngine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	var sessionBuffer *capture.TypingSessionBuffer
	var stopIdleChecker chan struct{}

	if enableKeylogger {
		// Create session buffer that flushes to database
		sessionBuffer = capture.NewTypingSessionBuffer(func(session *capture.TypingSessionData) {
			if session.Text == "" {
				return
			}
			ts := &storage.TypingSession{
				StartTime:         session.StartTime,
				EndTime:           session.EndTime,
				Text:              session.Text,
				KeyCount:          session.KeyCount,
				ActiveWindowTitle: session.Window,
				ActiveApp:         session.App,
			}
			if _, err := db.InsertTypingSession(ts); err != nil {
				log.Printf("Failed to insert typing session: %v", err)
			} else {
				log.Printf("Typing session saved: %q (%d keys, %s)", truncate(session.Text, 50), session.KeyCount, session.App)
			}
		})

		// Start idle checker to flush sessions after 30s of inactivity
		stopIdleChecker = sessionBuffer.StartIdleChecker(5 * time.Second)

		keylogger := capture.NewKeylogger(func(event capture.KeyEvent) {
			if event.State != capture.KeyStateDown {
				return
			}
			windowInfo, _ := capture.GetActiveWindow()
			app := ""
			window := ""
			if windowInfo != nil {
				app = windowInfo.App
				window = windowInfo.Title
			}
			sessionBuffer.AddKey(event.Key, app, window)
		})
		if err := keylogger.Start(); err != nil {
			log.Printf("Warning: Failed to start keylogger (may need accessibility permissions): %v", err)
		} else {
			log.Println("Keylogger started")
			defer func() {
				keylogger.Stop()
				if stopIdleChecker != nil {
					close(stopIdleChecker)
				}
				if sessionBuffer != nil {
					sessionBuffer.Flush() // Flush any remaining session
				}
			}()
		}
	}

	screenshotTicker := time.NewTicker(time.Duration(screenshotInterval) * time.Second)
	defer screenshotTicker.Stop()

	ocrTicker := time.NewTicker(60 * time.Minute)
	defer ocrTicker.Stop()

	captureScreenshot := func() {
		windowInfo, _ := capture.GetActiveWindow()

		filepath := fm.GetScreenshotPath(time.Now())
		if err := fm.EnsureDir(filepath); err != nil {
			log.Printf("Failed to create directory: %v", err)
			return
		}

		result, err := screenshotCapture.CaptureToFile(filepath)
		if err != nil {
			log.Printf("Failed to capture screenshot: %v", err)
			return
		}

		screenshot := &storage.Screenshot{
			Timestamp: result.Timestamp,
			Filepath:  filepath,
			Width:     result.Width,
			Height:    result.Height,
			FileSize:  int64(len(result.Data)),
		}
		if windowInfo != nil {
			screenshot.ActiveWindowTitle = windowInfo.Title
			screenshot.ActiveApp = windowInfo.App
		}

		if _, err := db.InsertScreenshot(screenshot); err != nil {
			log.Printf("Failed to insert screenshot: %v", err)
		} else {
			log.Printf("Captured screenshot: %s (%dx%d)", filepath, result.Width, result.Height)
		}
	}

	processOCR := func() {
		if !enableOCR {
			return
		}

		screenshots, err := db.GetUnprocessedScreenshots(50)
		if err != nil {
			log.Printf("Failed to get unprocessed screenshots: %v", err)
			return
		}

		for _, s := range screenshots {
			text, err := ocrEngine.ExtractText(s.Filepath)
			if err != nil {
				log.Printf("OCR failed for %s: %v", s.Filepath, err)
				continue
			}
			if err := db.UpdateScreenshotOCR(s.ID, text); err != nil {
				log.Printf("Failed to update OCR: %v", err)
			} else {
				log.Printf("OCR processed: %s (%d chars)", s.Filepath, len(text))
			}
		}
	}

	log.Println("Taking initial screenshot...")
	captureScreenshot()

	log.Printf("Daemon running. Screenshot interval: %ds", screenshotInterval)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-screenshotTicker.C:
			captureScreenshot()
		case <-ocrTicker.C:
			processOCR()
		}
	}
}
