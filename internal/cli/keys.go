package cli

import (
	"fmt"
	"time"

	"github.com/mahirisikli/memento/internal/storage"
	"github.com/spf13/cobra"
)

var (
	keysFrom   string
	keysTo     string
	keysApp    string
	keysToday  bool
	keysLimit  int
	keysSearch string
)

func init() {
	keysCmd.Flags().StringVar(&keysFrom, "from", "", "Start time")
	keysCmd.Flags().StringVar(&keysTo, "to", "", "End time")
	keysCmd.Flags().StringVar(&keysApp, "app", "", "Filter by application")
	keysCmd.Flags().BoolVar(&keysToday, "today", false, "Show today's typing sessions")
	keysCmd.Flags().IntVar(&keysLimit, "limit", 100, "Maximum sessions to return")
	keysCmd.Flags().StringVar(&keysSearch, "search", "", "Search text in typing sessions")
}

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "View typing sessions",
	Long:  `View logged typing sessions with optional filtering by time range, application, and text search.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		var from, to time.Time

		if keysToday {
			from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			to = now
		} else {
			from, to = parseTimeRange(keysFrom, keysTo)
			if keysFrom == "" && keysTo == "" {
				from = now.Add(-24 * time.Hour)
				to = now
			}
		}

		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		var sessions []storage.TypingSession
		if keysSearch != "" {
			sessions, err = db.SearchTypingSessions(keysSearch, from, to, keysLimit)
		} else {
			sessions, err = db.GetTypingSessionsByDateRange(from, to, keysApp, keysLimit)
		}
		if err != nil {
			return fmt.Errorf("failed to get typing sessions: %w", err)
		}

		format := getOutputFormat()
		switch format {
		case "json":
			output := map[string]interface{}{
				"from":     from,
				"to":       to,
				"app":      keysApp,
				"count":    len(sessions),
				"sessions": sessions,
			}
			outputJSON(output)
		case "plain":
			headers := []string{"id", "start", "end", "app", "window", "keys", "text"}
			var rows [][]string
			for _, s := range sessions {
				rows = append(rows, []string{
					fmt.Sprintf("%d", s.ID),
					s.StartTime.Format(time.RFC3339),
					s.EndTime.Format(time.RFC3339),
					s.ActiveApp,
					s.ActiveWindowTitle,
					fmt.Sprintf("%d", s.KeyCount),
					s.Text,
				})
			}
			outputPlain(headers, rows)
		default:
			if len(sessions) == 0 {
				fmt.Println("No typing sessions found.")
				return nil
			}

			// Calculate totals
			totalKeys := 0
			appKeys := make(map[string]int)
			for _, s := range sessions {
				totalKeys += s.KeyCount
				appKeys[s.ActiveApp] += s.KeyCount
			}

			fmt.Printf("Typing Sessions (%s to %s)\n", from.Format("2006-01-02 15:04"), to.Format("15:04"))
			fmt.Printf("Sessions: %d | Total keystrokes: %d\n\n", len(sessions), totalKeys)

			for _, s := range sessions {
				duration := s.EndTime.Sub(s.StartTime).Round(time.Second)
				fmt.Printf("[%s] %s - %s (%d keys, %s)\n",
					s.StartTime.Format("15:04:05"),
					s.ActiveApp,
					s.ActiveWindowTitle,
					s.KeyCount,
					duration,
				)
				text := s.Text
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				fmt.Printf("  %s\n\n", text)
			}
		}
		return nil
	},
}
