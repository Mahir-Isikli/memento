package cli

import (
	"fmt"
	"time"

	"github.com/mahirisikli/memento/internal/capture"
	"github.com/mahirisikli/memento/internal/ocr"
	"github.com/mahirisikli/memento/internal/storage"
	"github.com/spf13/cobra"
)

var (
	captureQuality    int
	captureFullScreen bool
	captureOCR        bool
)

func init() {
	captureCmd.Flags().IntVar(&captureQuality, "quality", 80, "WebP quality (1-100)")
	captureCmd.Flags().BoolVar(&captureFullScreen, "fullscreen", false, "Capture full screen")
	captureCmd.Flags().BoolVar(&captureOCR, "ocr", false, "Run OCR immediately")
}

var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Take a single screenshot now",
	Long:  `Capture a screenshot immediately without running the daemon.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		storagePath := getStoragePath()

		db, err := storage.NewDB(storagePath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		fm := storage.NewFileManager(storagePath)
		screenshotCapture := capture.NewScreenshotCapture(captureQuality, captureFullScreen)

		windowInfo, _ := capture.GetActiveWindow()

		filepath := fm.GetScreenshotPath(time.Now())
		if err := fm.EnsureDir(filepath); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		result, err := screenshotCapture.CaptureToFile(filepath)
		if err != nil {
			return fmt.Errorf("failed to capture screenshot: %w", err)
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

		id, err := db.InsertScreenshot(screenshot)
		if err != nil {
			return fmt.Errorf("failed to save screenshot: %w", err)
		}

		if captureOCR {
			ocrEngine := ocr.NewOCREngine()
			text, err := ocrEngine.ExtractText(filepath)
			if err != nil {
				fmt.Printf("Warning: OCR failed: %v\n", err)
			} else {
				db.UpdateScreenshotOCR(id, text)
				screenshot.OCRText = text
			}
		}

		format := getOutputFormat()
		switch format {
		case "json":
			outputJSON(map[string]interface{}{
				"id":         id,
				"filepath":   filepath,
				"width":      result.Width,
				"height":     result.Height,
				"size":       len(result.Data),
				"app":        screenshot.ActiveApp,
				"window":     screenshot.ActiveWindowTitle,
				"ocr_text":   screenshot.OCRText,
			})
		case "plain":
			fmt.Printf("%d\t%s\t%s\n", id, filepath, screenshot.ActiveApp)
		default:
			fmt.Printf("Screenshot captured!\n")
			fmt.Printf("  ID:       %d\n", id)
			fmt.Printf("  Path:     %s\n", filepath)
			fmt.Printf("  Size:     %dx%d (%d bytes)\n", result.Width, result.Height, len(result.Data))
			fmt.Printf("  App:      %s\n", screenshot.ActiveApp)
			fmt.Printf("  Window:   %s\n", screenshot.ActiveWindowTitle)
			if screenshot.OCRText != "" {
				snippet := screenshot.OCRText
				if len(snippet) > 100 {
					snippet = snippet[:100] + "..."
				}
				fmt.Printf("  OCR:      %s\n", snippet)
			}
		}

		return nil
	},
}
