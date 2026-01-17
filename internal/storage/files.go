package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileManager struct {
	basePath string
}

func NewFileManager(basePath string) *FileManager {
	return &FileManager{basePath: basePath}
}

func (fm *FileManager) GetScreenshotPath(t time.Time) string {
	dir := filepath.Join(
		fm.basePath,
		"screenshots",
		fmt.Sprintf("%d", t.Year()),
		fmt.Sprintf("%02d", t.Month()),
		fmt.Sprintf("%02d", t.Day()),
	)
	filename := fmt.Sprintf("%s.webp", t.Format("2006-01-02_15-04-05"))
	return filepath.Join(dir, filename)
}

func (fm *FileManager) EnsureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

func (fm *FileManager) GetLogsPath() string {
	return filepath.Join(fm.basePath, "logs")
}

func (fm *FileManager) EnsureLogsDir() error {
	return os.MkdirAll(fm.GetLogsPath(), 0755)
}

func (fm *FileManager) GetLogFilePath() string {
	return filepath.Join(fm.GetLogsPath(), "memento.log")
}

func (fm *FileManager) GetBasePath() string {
	return fm.basePath
}
