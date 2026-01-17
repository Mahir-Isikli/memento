package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type Config struct {
	ScreenshotIntervalSeconds int           `json:"screenshot_interval_seconds"`
	ScreenshotQuality         int           `json:"screenshot_quality"`
	CaptureFullScreen         bool          `json:"capture_full_screen"`
	OCRBatchIntervalMinutes   int           `json:"ocr_batch_interval_minutes"`
	Backup                    BackupConfig  `json:"backup"`
	StoragePath               string        `json:"storage_path"`
}

type BackupConfig struct {
	Enabled    bool   `json:"enabled"`
	Schedule   string `json:"schedule"`
	R2Bucket   string `json:"r2_bucket"`
	R2Endpoint string `json:"r2_endpoint"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		ScreenshotIntervalSeconds: 600,
		ScreenshotQuality:         80,
		CaptureFullScreen:         false,
		OCRBatchIntervalMinutes:   60,
		Backup: BackupConfig{
			Enabled:  false,
			Schedule: "daily",
		},
		StoragePath: filepath.Join(home, ".memento"),
	}
}

func LoadConfig() (*Config, error) {
	configPath := filepath.Join(getStoragePath(), "config.json")
	
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, err
	}
	
	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	return config, nil
}

func SaveConfig(config *Config) error {
	configPath := filepath.Join(getStoragePath(), "config.json")
	
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		format := getOutputFormat()
		switch format {
		case "json", "plain":
			outputJSON(config)
		default:
			fmt.Println("Memento Configuration")
			fmt.Println("=====================")
			fmt.Printf("Screenshot Interval: %d seconds\n", config.ScreenshotIntervalSeconds)
			fmt.Printf("Screenshot Quality:  %d%%\n", config.ScreenshotQuality)
			fmt.Printf("Capture Full Screen: %v\n", config.CaptureFullScreen)
			fmt.Printf("OCR Batch Interval:  %d minutes\n", config.OCRBatchIntervalMinutes)
			fmt.Printf("Storage Path:        %s\n", config.StoragePath)
			fmt.Println()
			fmt.Println("Backup:")
			fmt.Printf("  Enabled:  %v\n", config.Backup.Enabled)
			fmt.Printf("  Schedule: %s\n", config.Backup.Schedule)
			fmt.Printf("  R2 Bucket: %s\n", config.Backup.R2Bucket)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		
		config, err := LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		switch key {
		case "interval":
			var v int
			fmt.Sscanf(value, "%d", &v)
			config.ScreenshotIntervalSeconds = v
		case "quality":
			var v int
			fmt.Sscanf(value, "%d", &v)
			config.ScreenshotQuality = v
		case "fullscreen":
			config.CaptureFullScreen = value == "true" || value == "1"
		case "ocr_interval":
			var v int
			fmt.Sscanf(value, "%d", &v)
			config.OCRBatchIntervalMinutes = v
		case "backup_enabled":
			config.Backup.Enabled = value == "true" || value == "1"
		case "r2_bucket":
			config.Backup.R2Bucket = value
		case "r2_endpoint":
			config.Backup.R2Endpoint = value
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}
		
		if err := SaveConfig(config); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		
		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}
