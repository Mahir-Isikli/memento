package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
	path string
}

type Screenshot struct {
	ID                int64     `json:"id"`
	Timestamp         time.Time `json:"timestamp"`
	Filepath          string    `json:"filepath"`
	Width             int       `json:"width,omitempty"`
	Height            int       `json:"height,omitempty"`
	FileSize          int64     `json:"file_size,omitempty"`
	OCRText           string    `json:"ocr_text,omitempty"`
	OCRProcessedAt    *time.Time `json:"ocr_processed_at,omitempty"`
	ActiveWindowTitle string    `json:"active_window_title,omitempty"`
	ActiveApp         string    `json:"active_app,omitempty"`
}

type Keystroke struct {
	ID                int64     `json:"id"`
	Timestamp         time.Time `json:"timestamp"`
	Key               string    `json:"key"`
	Modifiers         []string  `json:"modifiers,omitempty"`
	ActiveWindowTitle string    `json:"active_window_title,omitempty"`
	ActiveApp         string    `json:"active_app,omitempty"`
}

func NewDB(storagePath string) (*DB, error) {
	dbPath := filepath.Join(storagePath, "memento.db")
	
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	db := &DB{conn: conn, path: dbPath}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	
	return db, nil
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS screenshots (
		id INTEGER PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		filepath TEXT NOT NULL,
		width INTEGER,
		height INTEGER,
		file_size INTEGER,
		ocr_text TEXT,
		ocr_processed_at DATETIME,
		active_window_title TEXT,
		active_app TEXT
	);
	
	CREATE TABLE IF NOT EXISTS keystrokes (
		id INTEGER PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		key TEXT NOT NULL,
		modifiers TEXT,
		active_window_title TEXT,
		active_app TEXT
	);
	
	CREATE INDEX IF NOT EXISTS idx_screenshots_timestamp ON screenshots(timestamp);
	CREATE INDEX IF NOT EXISTS idx_keystrokes_timestamp ON keystrokes(timestamp);
	CREATE INDEX IF NOT EXISTS idx_screenshots_ocr ON screenshots(ocr_text);
	`
	
	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) InsertScreenshot(s *Screenshot) (int64, error) {
	result, err := db.conn.Exec(`
		INSERT INTO screenshots (timestamp, filepath, width, height, file_size, active_window_title, active_app)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, s.Timestamp, s.Filepath, s.Width, s.Height, s.FileSize, s.ActiveWindowTitle, s.ActiveApp)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) UpdateScreenshotOCR(id int64, ocrText string) error {
	_, err := db.conn.Exec(`
		UPDATE screenshots SET ocr_text = ?, ocr_processed_at = ? WHERE id = ?
	`, ocrText, time.Now(), id)
	return err
}

func (db *DB) InsertKeystroke(k *Keystroke) (int64, error) {
	modifiersJSON, _ := json.Marshal(k.Modifiers)
	result, err := db.conn.Exec(`
		INSERT INTO keystrokes (timestamp, key, modifiers, active_window_title, active_app)
		VALUES (?, ?, ?, ?, ?)
	`, k.Timestamp, k.Key, string(modifiersJSON), k.ActiveWindowTitle, k.ActiveApp)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) SearchScreenshots(query string, from, to time.Time, limit int) ([]Screenshot, error) {
	if limit <= 0 {
		limit = 100
	}
	
	rows, err := db.conn.Query(`
		SELECT id, timestamp, filepath, width, height, file_size, ocr_text, ocr_processed_at, active_window_title, active_app
		FROM screenshots
		WHERE (ocr_text LIKE ? OR active_window_title LIKE ? OR active_app LIKE ?)
		AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, "%"+query+"%", "%"+query+"%", "%"+query+"%", from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []Screenshot
	for rows.Next() {
		var s Screenshot
		var ocrProcessedAt sql.NullTime
		err := rows.Scan(&s.ID, &s.Timestamp, &s.Filepath, &s.Width, &s.Height, &s.FileSize, &s.OCRText, &ocrProcessedAt, &s.ActiveWindowTitle, &s.ActiveApp)
		if err != nil {
			return nil, err
		}
		if ocrProcessedAt.Valid {
			s.OCRProcessedAt = &ocrProcessedAt.Time
		}
		results = append(results, s)
	}
	return results, nil
}

func (db *DB) GetScreenshotsByDateRange(from, to time.Time, limit int) ([]Screenshot, error) {
	if limit <= 0 {
		limit = 1000
	}
	
	rows, err := db.conn.Query(`
		SELECT id, timestamp, filepath, width, height, file_size, ocr_text, ocr_processed_at, active_window_title, active_app
		FROM screenshots
		WHERE timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []Screenshot
	for rows.Next() {
		var s Screenshot
		var ocrProcessedAt sql.NullTime
		err := rows.Scan(&s.ID, &s.Timestamp, &s.Filepath, &s.Width, &s.Height, &s.FileSize, &s.OCRText, &ocrProcessedAt, &s.ActiveWindowTitle, &s.ActiveApp)
		if err != nil {
			return nil, err
		}
		if ocrProcessedAt.Valid {
			s.OCRProcessedAt = &ocrProcessedAt.Time
		}
		results = append(results, s)
	}
	return results, nil
}

func (db *DB) GetUnprocessedScreenshots(limit int) ([]Screenshot, error) {
	if limit <= 0 {
		limit = 100
	}
	
	rows, err := db.conn.Query(`
		SELECT id, timestamp, filepath, width, height, file_size, active_window_title, active_app
		FROM screenshots
		WHERE ocr_processed_at IS NULL
		ORDER BY timestamp ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []Screenshot
	for rows.Next() {
		var s Screenshot
		err := rows.Scan(&s.ID, &s.Timestamp, &s.Filepath, &s.Width, &s.Height, &s.FileSize, &s.ActiveWindowTitle, &s.ActiveApp)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func (db *DB) GetKeystrokesByDateRange(from, to time.Time, app string, limit int) ([]Keystroke, error) {
	if limit <= 0 {
		limit = 10000
	}
	
	var rows *sql.Rows
	var err error
	
	if app != "" {
		rows, err = db.conn.Query(`
			SELECT id, timestamp, key, modifiers, active_window_title, active_app
			FROM keystrokes
			WHERE timestamp BETWEEN ? AND ? AND active_app LIKE ?
			ORDER BY timestamp ASC
			LIMIT ?
		`, from, to, "%"+app+"%", limit)
	} else {
		rows, err = db.conn.Query(`
			SELECT id, timestamp, key, modifiers, active_window_title, active_app
			FROM keystrokes
			WHERE timestamp BETWEEN ? AND ?
			ORDER BY timestamp ASC
			LIMIT ?
		`, from, to, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []Keystroke
	for rows.Next() {
		var k Keystroke
		var modifiersJSON string
		err := rows.Scan(&k.ID, &k.Timestamp, &k.Key, &modifiersJSON, &k.ActiveWindowTitle, &k.ActiveApp)
		if err != nil {
			return nil, err
		}
		if modifiersJSON != "" {
			json.Unmarshal([]byte(modifiersJSON), &k.Modifiers)
		}
		results = append(results, k)
	}
	return results, nil
}

func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	var screenshotCount int64
	db.conn.QueryRow("SELECT COUNT(*) FROM screenshots").Scan(&screenshotCount)
	stats["screenshot_count"] = screenshotCount
	
	var keystrokeCount int64
	db.conn.QueryRow("SELECT COUNT(*) FROM keystrokes").Scan(&keystrokeCount)
	stats["keystroke_count"] = keystrokeCount
	
	var ocrProcessed int64
	db.conn.QueryRow("SELECT COUNT(*) FROM screenshots WHERE ocr_processed_at IS NOT NULL").Scan(&ocrProcessed)
	stats["ocr_processed"] = ocrProcessed
	
	var oldestScreenshot sql.NullTime
	db.conn.QueryRow("SELECT MIN(timestamp) FROM screenshots").Scan(&oldestScreenshot)
	if oldestScreenshot.Valid {
		stats["oldest_screenshot"] = oldestScreenshot.Time
	}
	
	var newestScreenshot sql.NullTime
	db.conn.QueryRow("SELECT MAX(timestamp) FROM screenshots").Scan(&newestScreenshot)
	if newestScreenshot.Valid {
		stats["newest_screenshot"] = newestScreenshot.Time
	}
	
	return stats, nil
}
