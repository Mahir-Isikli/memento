package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func runSemantic(mode, query string, from, to time.Time, limit int) error {
	home, _ := os.UserHomeDir()
	runtime := filepath.Join(home, ".local", "share", "memento", "semantic")
	if p := os.Getenv("MEMENTO_SEMANTIC_RUNTIME"); p != "" {
		runtime = p
	}
	script := filepath.Join(runtime, "search.py")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("semantic runtime missing at %s", script)
	}
	args := []string{"run", "--offline", "--no-sync", "--project", runtime, "python", script, mode, "--storage", getStoragePath()}
	if mode == "search" {
		args = append(args, "--query", query, "--from", from.Format(time.RFC3339), "--to", to.Format(time.RFC3339), "--limit", fmt.Sprint(limit))
	}
	if getOutputFormat() == "json" {
		args = append(args, "--json")
	}
	cmd := exec.Command("uv", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	semantic := &cobra.Command{Use: "semantic", Short: "Manage the small local semantic search model and index"}
	for _, mode := range []string{"setup", "index", "status"} {
		semantic.AddCommand(&cobra.Command{Use: mode, RunE: func(cmd *cobra.Command, args []string) error {
			return runSemantic(cmd.Name(), "", time.Time{}, time.Time{}, 10)
		}})
	}
	rootCmd.AddCommand(semantic)
}
