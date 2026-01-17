package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	outputFormat string
	storagePath  string
)

func getStoragePath() string {
	if storagePath != "" {
		return storagePath
	}
	if p := os.Getenv("MEMENTO_STORAGE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".memento")
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text, json, plain")
	rootCmd.PersistentFlags().StringVar(&storagePath, "storage", "", "Override storage path")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(timelineCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(screenshotsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(captureCmd)
}

var rootCmd = &cobra.Command{
	Use:   "memento",
	Short: "Personal digital memory archive",
	Long: `Memento captures screenshots every 10 minutes, logs keystrokes with active window context,
runs OCR on captured images, and provides an agent-friendly CLI for querying your digital history.

Inspired by Tobi Lutke's 15-year personal archiving setup.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func outputJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

func outputPlain(headers []string, rows [][]string) {
	fmt.Println(strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Println(strings.Join(row, "\t"))
	}
}

func getOutputFormat() string {
	if outputFormat != "" && outputFormat != "text" {
		return outputFormat
	}
	if os.Getenv("MEMENTO_JSON") != "" {
		return "json"
	}
	if os.Getenv("MEMENTO_PLAIN") != "" {
		return "plain"
	}
	return "text"
}
