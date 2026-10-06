package storage

import "time"

type Dictation struct {
	Source        string    `json:"source"`
	SourceID      string    `json:"source_id"`
	Timestamp     time.Time `json:"timestamp"`
	RawText       string    `json:"raw_text"`
	FormattedText string    `json:"formatted_text"`
	EditedText    string    `json:"edited_text"`
	PastedText    string    `json:"pasted_text"`
	ActiveApp     string    `json:"app"`
	URL           string    `json:"url,omitempty"`
	Duration      float64   `json:"duration_seconds"`
}

func (d Dictation) Text() string {
	for _, text := range []string{d.EditedText, d.PastedText, d.FormattedText, d.RawText} {
		if text != "" {
			return text
		}
	}
	return ""
}

func (db *DB) UpsertDictation(d Dictation) error {
	_, err := db.conn.Exec(`INSERT INTO dictations
		(source,source_id,timestamp,raw_text,formatted_text,edited_text,pasted_text,active_app,url,duration)
		VALUES (?,?,?,?,?,?,?,?,?,?) ON CONFLICT(source,source_id) DO UPDATE SET
		timestamp=excluded.timestamp,raw_text=excluded.raw_text,formatted_text=excluded.formatted_text,
		edited_text=excluded.edited_text,pasted_text=excluded.pasted_text,active_app=excluded.active_app,
		url=excluded.url,duration=excluded.duration`, d.Source, d.SourceID, d.Timestamp, d.RawText,
		d.FormattedText, d.EditedText, d.PastedText, d.ActiveApp, d.URL, d.Duration)
	return err
}

func (db *DB) SearchDictations(query string, from, to time.Time, limit int) ([]Dictation, error) {
	rows, err := db.conn.Query(`SELECT source,source_id,timestamp,raw_text,formatted_text,edited_text,
		pasted_text,active_app,url,duration FROM dictations WHERE timestamp BETWEEN ? AND ?
		AND (raw_text LIKE ? OR formatted_text LIKE ? OR edited_text LIKE ? OR pasted_text LIKE ? OR active_app LIKE ?)
		ORDER BY timestamp DESC LIMIT ?`, from, to, "%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []Dictation{}
	for rows.Next() {
		var d Dictation
		if err := rows.Scan(&d.Source, &d.SourceID, &d.Timestamp, &d.RawText, &d.FormattedText, &d.EditedText,
			&d.PastedText, &d.ActiveApp, &d.URL, &d.Duration); err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, rows.Err()
}
