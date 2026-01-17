package cli

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/mahirisikli/memento/internal/storage"
	"github.com/spf13/cobra"
)

var (
	screenshotsToday bool
	screenshotsDate  string
	screenshotsLimit int
)

func init() {
	screenshotsListCmd.Flags().BoolVar(&screenshotsToday, "today", false, "Show today's screenshots")
	screenshotsListCmd.Flags().StringVar(&screenshotsDate, "date", "", "Specific date")
	screenshotsListCmd.Flags().IntVar(&screenshotsLimit, "limit", 100, "Maximum results")

	screenshotsCmd.AddCommand(screenshotsListCmd)
	screenshotsCmd.AddCommand(screenshotsShowCmd)
	screenshotsCmd.AddCommand(screenshotsOCRCmd)
}

var screenshotsCmd = &cobra.Command{
	Use:   "screenshots",
	Short: "Manage screenshots",
	Long:  `List, view, and manage captured screenshots.`,
}

var screenshotsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List screenshots",
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		var from, to time.Time

		if screenshotsToday || screenshotsDate == "" {
			from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			to = now
		} else if screenshotsDate != "" {
			if parsed := parseRelativeTime(screenshotsDate, now); !parsed.IsZero() {
				from = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, parsed.Location())
				to = from.AddDate(0, 0, 1)
			}
		}

		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		results, err := db.GetScreenshotsByDateRange(from, to, screenshotsLimit)
		if err != nil {
			return fmt.Errorf("failed to list screenshots: %w", err)
		}

		format := getOutputFormat()
		switch format {
		case "json":
			outputJSON(results)
		case "plain":
			headers := []string{"id", "timestamp", "app", "filepath", "ocr_processed"}
			var rows [][]string
			for _, r := range results {
				ocrStatus := "no"
				if r.OCRProcessedAt != nil {
					ocrStatus = "yes"
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", r.ID),
					r.Timestamp.Format(time.RFC3339),
					r.ActiveApp,
					r.Filepath,
					ocrStatus,
				})
			}
			outputPlain(headers, rows)
		default:
			if len(results) == 0 {
				fmt.Println("No screenshots found.")
				return nil
			}
			fmt.Printf("Screenshots (%d)\n\n", len(results))
			for _, r := range results {
				ocrStatus := ""
				if r.OCRProcessedAt != nil {
					ocrStatus = " [OCR]"
				}
				fmt.Printf("[%d] %s - %s%s\n", r.ID, r.Timestamp.Format("15:04:05"), r.ActiveApp, ocrStatus)
				fmt.Printf("     %s\n", r.Filepath)
			}
		}
		return nil
	},
}

var screenshotsShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Open a screenshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var id int64
		fmt.Sscanf(args[0], "%d", &id)

		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		// Get screenshot by searching with empty query
		results, err := db.GetScreenshotsByDateRange(time.Time{}, time.Now().AddDate(100, 0, 0), 10000)
		if err != nil {
			return fmt.Errorf("failed to find screenshot: %w", err)
		}

		for _, r := range results {
			if r.ID == id {
				// Open with default viewer
				return exec.Command("open", r.Filepath).Run()
			}
		}

		return fmt.Errorf("screenshot with ID %d not found", id)
	},
}

var screenshotsOCRCmd = &cobra.Command{
	Use:   "ocr [id]",
	Short: "Show OCR text for a screenshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var id int64
		fmt.Sscanf(args[0], "%d", &id)

		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		results, err := db.GetScreenshotsByDateRange(time.Time{}, time.Now().AddDate(100, 0, 0), 10000)
		if err != nil {
			return fmt.Errorf("failed to find screenshot: %w", err)
		}

		for _, r := range results {
			if r.ID == id {
				format := getOutputFormat()
				switch format {
				case "json":
					outputJSON(map[string]interface{}{
						"id":        r.ID,
						"filepath":  r.Filepath,
						"ocr_text":  r.OCRText,
						"processed": r.OCRProcessedAt,
					})
				default:
					if r.OCRText == "" {
						fmt.Println("No OCR text available for this screenshot.")
					} else {
						fmt.Println(r.OCRText)
					}
				}
				return nil
			}
		}

		return fmt.Errorf("screenshot with ID %d not found", id)
	},
}
