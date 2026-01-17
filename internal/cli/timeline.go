package cli

import (
	"fmt"
	"time"

	"github.com/mahirisikli/memento/internal/storage"
	"github.com/spf13/cobra"
)

var (
	timelineDate  string
	timelineFrom  string
	timelineTo    string
	timelineLimit int
)

func init() {
	timelineCmd.Flags().StringVar(&timelineDate, "date", "", "Specific date (e.g., 2026-01-15)")
	timelineCmd.Flags().StringVar(&timelineFrom, "from", "", "Start date")
	timelineCmd.Flags().StringVar(&timelineTo, "to", "", "End date")
	timelineCmd.Flags().IntVar(&timelineLimit, "limit", 100, "Maximum results")
}

var timelineCmd = &cobra.Command{
	Use:   "timeline",
	Short: "Browse activity timeline",
	Long:  `View a timeline of captured screenshots and activity for a specific date or range.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		var from, to time.Time

		if timelineDate != "" {
			if parsed := parseRelativeTime(timelineDate, now); !parsed.IsZero() {
				from = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, parsed.Location())
				to = from.AddDate(0, 0, 1)
			}
		} else {
			from, to = parseTimeRange(timelineFrom, timelineTo)
			if timelineFrom == "" && timelineTo == "" {
				// Default to today
				from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				to = now
			}
		}

		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		results, err := db.GetScreenshotsByDateRange(from, to, timelineLimit)
		if err != nil {
			return fmt.Errorf("failed to get timeline: %w", err)
		}

		format := getOutputFormat()
		switch format {
		case "json":
			output := map[string]interface{}{
				"from":    from,
				"to":      to,
				"count":   len(results),
				"results": results,
			}
			outputJSON(output)
		case "plain":
			headers := []string{"id", "timestamp", "app", "window", "screenshot"}
			var rows [][]string
			for _, r := range results {
				rows = append(rows, []string{
					fmt.Sprintf("%d", r.ID),
					r.Timestamp.Format(time.RFC3339),
					r.ActiveApp,
					r.ActiveWindowTitle,
					r.Filepath,
				})
			}
			outputPlain(headers, rows)
		default:
			if len(results) == 0 {
				fmt.Printf("No activity found for %s to %s\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
				return nil
			}
			fmt.Printf("Timeline: %s to %s (%d entries)\n\n", from.Format("2006-01-02 15:04"), to.Format("2006-01-02 15:04"), len(results))
			
			currentHour := -1
			for _, r := range results {
				hour := r.Timestamp.Hour()
				if hour != currentHour {
					fmt.Printf("\n--- %02d:00 ---\n", hour)
					currentHour = hour
				}
				fmt.Printf("[%s] %s - %s\n", r.Timestamp.Format("15:04:05"), r.ActiveApp, r.ActiveWindowTitle)
			}
		}
		return nil
	},
}
