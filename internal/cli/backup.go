package cli

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage backups to R2",
}

var backupNowCmd = &cobra.Command{
	Use:   "now",
	Short: "Trigger an immediate backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if !config.Backup.Enabled {
			return fmt.Errorf("backup is not enabled. Run: memento config set backup_enabled true")
		}

		if config.Backup.R2Bucket == "" {
			return fmt.Errorf("R2 bucket not configured. Run: memento config set r2_bucket <bucket-name>")
		}

		storagePath := getStoragePath()
		remotePath := fmt.Sprintf("r2:%s/memento", config.Backup.R2Bucket)

		fmt.Printf("Backing up %s to %s...\n", storagePath, remotePath)

		rcloneCmd := exec.Command("rclone", "sync", storagePath, remotePath, 
			"--progress",
			"--exclude", "*.log",
		)
		rcloneCmd.Stdout = cmd.OutOrStdout()
		rcloneCmd.Stderr = cmd.ErrOrStderr()

		if err := rcloneCmd.Run(); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}

		fmt.Println("Backup completed successfully!")
		return nil
	},
}

var backupStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show backup status",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		format := getOutputFormat()
		status := map[string]interface{}{
			"enabled":     config.Backup.Enabled,
			"schedule":    config.Backup.Schedule,
			"r2_bucket":   config.Backup.R2Bucket,
			"r2_endpoint": config.Backup.R2Endpoint,
			"checked_at":  time.Now(),
		}

		// Check if rclone is available
		if _, err := exec.LookPath("rclone"); err != nil {
			status["rclone_installed"] = false
		} else {
			status["rclone_installed"] = true
		}

		switch format {
		case "json":
			outputJSON(status)
		default:
			fmt.Println("Backup Status")
			fmt.Println("=============")
			fmt.Printf("Enabled:     %v\n", config.Backup.Enabled)
			fmt.Printf("Schedule:    %s\n", config.Backup.Schedule)
			fmt.Printf("R2 Bucket:   %s\n", config.Backup.R2Bucket)
			if status["rclone_installed"].(bool) {
				fmt.Println("rclone:      installed")
			} else {
				fmt.Println("rclone:      NOT INSTALLED (brew install rclone)")
			}
		}
		return nil
	},
}

func init() {
	backupCmd.AddCommand(backupNowCmd)
	backupCmd.AddCommand(backupStatusCmd)
}
