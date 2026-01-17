package storage

import (
	"database/sql"
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

type TypingSession struct {
	ID                int64     `json:"id"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Text              string    `json:"text"`
	KeyCount          int       `json:"key_count"`
	ActiveWindowTitle string    `json:"window,omitempty"`
	ActiveApp         string    `json:"app,omitempty"`
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
	
	CREATE TABLE IF NOT EXISTS typing_sessions (
		id INTEGER PRIMARY KEY,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		text TEXT NOT NULL,
		key_count INTEGER NOT NULL,
		active_window_title TEXT,
		active_app TEXT
	);
	
	CREATE INDEX IF NOT EXISTS idx_screenshots_timestamp ON screenshots(timestamp);
	CREATE INDEX IF NOT EXISTS idx_typing_sessions_start ON typing_sessions(start_time);
	CREATE INDEX IF NOT EXISTS idx_typing_sessions_text ON typing_sessions(text);
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

func (db *DB) InsertTypingSession(s *TypingSession) (int64, error) {
	result, err := db.conn.Exec(`
		INSERT INTO typing_sessions (start_time, end_time, text, key_count, active_window_title, active_app)
		VALUES (?, ?, ?, ?, ?, ?)
	`, s.StartTime, s.EndTime, s.Text, s.KeyCount, s.ActiveWindowTitle, s.ActiveApp)
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
		var ocrText sql.NullString
		var ocrProcessedAt sql.NullTime
		err := rows.Scan(&s.ID, &s.Timestamp, &s.Filepath, &s.Width, &s.Height, &s.FileSize, &ocrText, &ocrProcessedAt, &s.ActiveWindowTitle, &s.ActiveApp)
		if err != nil {
			return nil, err
		}
		if ocrText.Valid {
			s.OCRText = ocrText.String
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
		var ocrText sql.NullString
		var ocrProcessedAt sql.NullTime
		err := rows.Scan(&s.ID, &s.Timestamp, &s.Filepath, &s.Width, &s.Height, &s.FileSize, &ocrText, &ocrProcessedAt, &s.ActiveWindowTitle, &s.ActiveApp)
		if err != nil {
			return nil, err
		}
		if ocrText.Valid {
			s.OCRText = ocrText.String
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

func (db *DB) GetTypingSessionsByDateRange(from, to time.Time, app string, limit int) ([]TypingSession, error) {
	if limit <= 0 {
		limit = 1000
	}
	
	var rows *sql.Rows
	var err error
	
	if app != "" {
		rows, err = db.conn.Query(`
			SELECT id, start_time, end_time, text, key_count, active_window_title, active_app
			FROM typing_sessions
			WHERE start_time BETWEEN ? AND ? AND active_app LIKE ?
			ORDER BY start_time DESC
			LIMIT ?
		`, from, to, "%"+app+"%", limit)
	} else {
		rows, err = db.conn.Query(`
			SELECT id, start_time, end_time, text, key_count, active_window_title, active_app
			FROM typing_sessions
			WHERE start_time BETWEEN ? AND ?
			ORDER BY start_time DESC
			LIMIT ?
		`, from, to, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []TypingSession
	for rows.Next() {
		var s TypingSession
		err := rows.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.Text, &s.KeyCount, &s.ActiveWindowTitle, &s.ActiveApp)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func (db *DB) SearchTypingSessions(query string, from, to time.Time, limit int) ([]TypingSession, error) {
	if limit <= 0 {
		limit = 100
	}
	
	rows, err := db.conn.Query(`
		SELECT id, start_time, end_time, text, key_count, active_window_title, active_app
		FROM typing_sessions
		WHERE text LIKE ? AND start_time BETWEEN ? AND ?
		ORDER BY start_time DESC
		LIMIT ?
	`, "%"+query+"%", from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []TypingSession
	for rows.Next() {
		var s TypingSession
		err := rows.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.Text, &s.KeyCount, &s.ActiveWindowTitle, &s.ActiveApp)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	var screenshotCount int64
	db.conn.QueryRow("SELECT COUNT(*) FROM screenshots").Scan(&screenshotCount)
	stats["screenshot_count"] = screenshotCount
	
	var sessionCount int64
	db.conn.QueryRow("SELECT COUNT(*) FROM typing_sessions").Scan(&sessionCount)
	stats["typing_session_count"] = sessionCount
	
	var totalKeystrokes int64
	db.conn.QueryRow("SELECT COALESCE(SUM(key_count), 0) FROM typing_sessions").Scan(&totalKeystrokes)
	stats["total_keystrokes"] = totalKeystrokes
	
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
