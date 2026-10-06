package cli

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mahirisikli/memento/internal/storage"
	"github.com/spf13/cobra"
)

func defaultWisprPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "Wispr Flow", "flow.sqlite")
}

func importWisprHistory(db *storage.DB, sourcePath string) (int, error) {
	uri := url.URL{Scheme: "file", Path: sourcePath}
	source, err := sql.Open("sqlite3", uri.String()+"?mode=ro&_busy_timeout=5000")
	if err != nil {
		return 0, err
	}
	defer source.Close()
	rows, err := source.Query(`SELECT transcriptEntityId, CAST(timestamp AS TEXT), COALESCE(asrText,''),
		COALESCE(formattedText,''),COALESCE(editedText,''),COALESCE(pastedText,''),
		COALESCE(app,''),COALESCE(url,''),COALESCE(duration,0) FROM History
		WHERE COALESCE(asrText,'') != '' OR COALESCE(formattedText,'') != ''
		OR COALESCE(editedText,'') != '' OR COALESCE(pastedText,'') != ''`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		d := storage.Dictation{Source: "wispr-flow"}
		var timestamp string
		if err := rows.Scan(&d.SourceID, &timestamp, &d.RawText, &d.FormattedText, &d.EditedText,
			&d.PastedText, &d.ActiveApp, &d.URL, &d.Duration); err != nil {
			return count, err
		}
		for _, layout := range []string{"2006-01-02 15:04:05.999999999 -07:00", time.RFC3339Nano, "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
			d.Timestamp, err = time.Parse(layout, timestamp)
			if err == nil {
				break
			}
		}
		if err != nil {
			return count, fmt.Errorf("invalid timestamp for Wispr entry %s: %w", d.SourceID, err)
		}
		if d.Timestamp.IsZero() {
			return count, fmt.Errorf("missing timestamp for Wispr entry %s", d.SourceID)
		}
		if err := db.UpsertDictation(d); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

var wisprCmd = &cobra.Command{Use: "wispr", Short: "Import local Wispr Flow dictation history"}
var wisprSourcePath string
var wisprSyncCmd = &cobra.Command{
	Use: "sync", Short: "Read local Wispr Flow history and upsert transcripts into Memento",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return err
		}
		defer db.Close()
		count, err := importWisprHistory(db, wisprSourcePath)
		if err != nil {
			return err
		}
		if getOutputFormat() == "json" {
			outputJSON(map[string]interface{}{"source": "wispr-flow", "checked": count})
		} else {
			fmt.Printf("Synced %d local Wispr Flow transcripts. Audio and screenshots were not copied.\n", count)
		}
		return nil
	},
}

var dictationFrom, dictationTo string
var dictationLimit int
var dictationsCmd = &cobra.Command{
	Use: "dictations [query]", Short: "Search imported voice transcripts",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB(getStoragePath())
		if err != nil {
			return err
		}
		defer db.Close()
		from, to := parseTimeRange(dictationFrom, dictationTo)
		results, err := db.SearchDictations(strings.Join(args, " "), from, to, dictationLimit)
		if err != nil {
			return err
		}
		if getOutputFormat() == "json" {
			outputJSON(map[string]interface{}{"count": len(results), "results": results})
		} else {
			for _, d := range results {
				fmt.Printf("[%s] %s (%s)\n%s\n\n", d.Timestamp.Local().Format("2006-01-02 15:04:05"), d.ActiveApp, d.Source, d.Text())
			}
		}
		return nil
	},
}

func init() {
	wisprSyncCmd.Flags().StringVar(&wisprSourcePath, "source", defaultWisprPath(), "Read-only source database")
	wisprCmd.AddCommand(wisprSyncCmd)
	dictationsCmd.Flags().StringVar(&dictationFrom, "from", "", "Start date")
	dictationsCmd.Flags().StringVar(&dictationTo, "to", "", "End date")
	dictationsCmd.Flags().IntVar(&dictationLimit, "limit", 100, "Maximum results")
	rootCmd.AddCommand(wisprCmd, dictationsCmd)
}
