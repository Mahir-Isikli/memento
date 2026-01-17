package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/mahirisikli/memento/internal/storage"
	"github.com/spf13/cobra"
)

var (
	keysFrom  string
	keysTo    string
	keysApp   string
	keysToday bool
	keysLimit int
)

func init() {
	keysCmd.Flags().StringVar(&keysFrom, "from", "", "Start time")
	keysCmd.Flags().StringVar(&keysTo, "to", "", "End time")
	keysCmd.Flags().StringVar(&keysApp, "app", "", "Filter by application")
	keysCmd.Flags().BoolVar(&keysToday, "today", false, "Show today's keystrokes")
	keysCmd.Flags().IntVar(&keysLimit, "limit", 10000, "Maximum keystrokes to return")
}

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "View keystroke history",
	Long:  `View logged keystrokes with optional filtering by time range and application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		var from, to time.Time

		if keysToday {
			from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			to = now
		} else {
			from, to = parseTimeRange(keysFrom, keysTo)
			if keysFrom == "" && keysTo == "" {
				// Default to last 2 hours
				from = now.Add(-2 * time.Hour)
				to = now
			}
		}

		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		keystrokes, err := db.GetKeystrokesByDateRange(from, to, keysApp, keysLimit)
		if err != nil {
			return fmt.Errorf("failed to get keystrokes: %w", err)
		}

		format := getOutputFormat()
		switch format {
		case "json":
			output := map[string]interface{}{
				"from":   from,
				"to":     to,
				"app":    keysApp,
				"count":  len(keystrokes),
				"keys":   keystrokes,
			}
			outputJSON(output)
		case "plain":
			headers := []string{"timestamp", "key", "modifiers", "app", "window"}
			var rows [][]string
			for _, k := range keystrokes {
				rows = append(rows, []string{
					k.Timestamp.Format(time.RFC3339),
					k.Key,
					strings.Join(k.Modifiers, "+"),
					k.ActiveApp,
					k.ActiveWindowTitle,
				})
			}
			outputPlain(headers, rows)
		default:
			if len(keystrokes) == 0 {
				fmt.Println("No keystrokes found.")
				return nil
			}
			
			// Group by app and show summary
			appCounts := make(map[string]int)
			for _, k := range keystrokes {
				appCounts[k.ActiveApp]++
			}
			
			fmt.Printf("Keystroke Summary (%s to %s)\n", from.Format("15:04"), to.Format("15:04"))
			fmt.Printf("Total keystrokes: %d\n\n", len(keystrokes))
			
			fmt.Println("By Application:")
			for app, count := range appCounts {
				fmt.Printf("  %-30s: %d\n", app, count)
			}
		}
		return nil
	},
}
