package cli

import (
	"fmt"
	"time"

	"github.com/mahirisikli/memento/internal/storage"
	"github.com/spf13/cobra"
)

var (
	searchFrom     string
	searchTo       string
	searchLimit    int
	semanticSearch bool
)

func init() {
	searchCmd.Flags().StringVar(&searchFrom, "from", "", "Start date (e.g., '2 days ago', '2026-01-15')")
	searchCmd.Flags().StringVar(&searchTo, "to", "", "End date (e.g., 'today', '2026-01-17')")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 100, "Maximum results to return")
	searchCmd.Flags().BoolVar(&semanticSearch, "semantic", false, "Search by meaning using the small local embedding model")
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search OCR text and window titles",
	Long:  `Search through captured screenshots by OCR text, window title, or application name.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		from, to := parseTimeRange(searchFrom, searchTo)
		if semanticSearch {
			if !cmd.Flags().Changed("limit") {
				searchLimit = 10
			}
			return runSemantic("search", query, from, to, searchLimit)
		}

		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		results, err := db.SearchScreenshots(query, from, to, searchLimit)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		dictations, err := db.SearchDictations(query, from, to, searchLimit)
		if err != nil {
			return err
		}
		sessions, err := db.SearchTypingSessions(query, from, to, searchLimit)
		if err != nil {
			return err
		}
		total := len(results) + len(dictations) + len(sessions)

		format := getOutputFormat()
		switch format {
		case "json":
			output := map[string]interface{}{
				"query":           query,
				"count":           total,
				"results":         results,
				"dictations":      dictations,
				"typing_sessions": sessions,
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
			for _, d := range dictations {
				fmt.Printf("dictation\t%s\t%s\t%s\n", d.Timestamp.Format(time.RFC3339), d.ActiveApp, d.SourceID)
			}
			for _, s := range sessions {
				fmt.Printf("typing\t%s\t%s\t%d\n", s.StartTime.Format(time.RFC3339), s.ActiveApp, s.ID)
			}
		default:
			if total == 0 {
				fmt.Println("No results found.")
				return nil
			}
			fmt.Printf("Found %d results for \"%s\":\n\n", total, query)
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
			for _, d := range dictations {
				fmt.Printf("[%s] Dictation in %s (%s)\n%s\n\n", d.Timestamp.Local().Format("2006-01-02 15:04:05"), d.ActiveApp, d.Source, d.Text())
			}
			for _, s := range sessions {
				fmt.Printf("[%s] Typed in %s\n%s\n\n", s.StartTime.Local().Format("2006-01-02 15:04:05"), s.ActiveApp, s.Text)
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
