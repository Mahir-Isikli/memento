package cli

import (
	"fmt"
	"time"

	"github.com/mahirisikli/memento/internal/storage"
	"github.com/spf13/cobra"
)

var (
	searchFrom  string
	searchTo    string
	searchLimit int
)

func init() {
	searchCmd.Flags().StringVar(&searchFrom, "from", "", "Start date (e.g., '2 days ago', '2026-01-15')")
	searchCmd.Flags().StringVar(&searchTo, "to", "", "End date (e.g., 'today', '2026-01-17')")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 100, "Maximum results to return")
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search OCR text and window titles",
	Long:  `Search through captured screenshots by OCR text, window title, or application name.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		from, to := parseTimeRange(searchFrom, searchTo)

		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		results, err := db.SearchScreenshots(query, from, to, searchLimit)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		format := getOutputFormat()
		switch format {
		case "json":
			output := map[string]interface{}{
				"query":   query,
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
				fmt.Println("No results found.")
				return nil
			}
			fmt.Printf("Found %d results for \"%s\":\n\n", len(results), query)
			for _, r := range results {
				fmt.Printf("[%s] %s - %s\n", r.Timestamp.Format("2006-01-02 15:04:05"), r.ActiveApp, r.ActiveWindowTitle)
				fmt.Printf("  Screenshot: %s\n", r.Filepath)
				if r.OCRText != "" {
					snippet := r.OCRText
					if len(snippet) > 100 {
						snippet = snippet[:100] + "..."
					}
					fmt.Printf("  OCR: %s\n", snippet)
				}
				fmt.Println()
			}
		}
		return nil
	},
}

func parseTimeRange(fromStr, toStr string) (time.Time, time.Time) {
	now := time.Now()
	from := now.AddDate(-1, 0, 0) // Default: 1 year ago
	to := now

	if fromStr != "" {
		if parsed := parseRelativeTime(fromStr, now); !parsed.IsZero() {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed := parseRelativeTime(toStr, now); !parsed.IsZero() {
			to = parsed
		}
	}

	return from, to
}

func parseRelativeTime(s string, now time.Time) time.Time {
	switch s {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "yesterday":
		return now.AddDate(0, 0, -1)
	case "1 day ago", "1day ago":
		return now.AddDate(0, 0, -1)
	case "2 days ago", "2days ago":
		return now.AddDate(0, 0, -2)
	case "1 week ago", "1week ago":
		return now.AddDate(0, 0, -7)
	case "2 weeks ago", "2weeks ago":
		return now.AddDate(0, 0, -14)
	case "1 month ago", "1month ago":
		return now.AddDate(0, -1, 0)
	case "1 hour ago", "1hour ago":
		return now.Add(-1 * time.Hour)
	case "2 hours ago", "2hours ago":
		return now.Add(-2 * time.Hour)
	}

	// Try parsing as date
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Time{}
}
