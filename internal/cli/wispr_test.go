package cli

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/mahirisikli/memento/internal/storage"
)

func TestWisprImportIsReadOnlyAndIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flow.sqlite")
	source, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = source.Exec(`CREATE TABLE History (transcriptEntityId TEXT,timestamp DATETIME,asrText TEXT,formattedText TEXT,editedText TEXT,pastedText TEXT,app TEXT,url TEXT,duration REAL);
	INSERT INTO History VALUES ('one','2026-10-05 08:10:00.123 +00:00','launch draft','Launch draft.',NULL,'Launch draft.','Notes',NULL,4);
	INSERT INTO History VALUES ('two','2026-10-05T08:11:00Z','','Only formatted.',NULL,NULL,'Notes',NULL,2);`)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	db, err := storage.NewDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 2; i++ {
		n, err := importWisprHistory(db, path)
		if err != nil || n != 2 {
			t.Fatalf("import: count=%d err=%v", n, err)
		}
	}
	stats, err := db.GetStats()
	if err != nil || stats["dictation_count"] != int64(2) {
		t.Fatalf("stats: %v %v", stats, err)
	}
	rows, err := db.SearchDictations("draft", time.Time{}, time.Now().AddDate(1, 0, 0), 10)
	if err != nil || len(rows) != 1 || rows[0].FormattedText != "Launch draft." {
		t.Fatalf("search: %v %v", rows, err)
	}
	_, err = source.Exec("UPDATE History SET editedText='Updated launch draft.' WHERE transcriptEntityId='one'")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := importWisprHistory(db, path); err != nil {
		t.Fatal(err)
	}
	rows, err = db.SearchDictations("Updated", time.Time{}, time.Now().AddDate(1, 0, 0), 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("updated search: %v %v", rows, err)
	}
	var text string
	if err := source.QueryRow("SELECT formattedText FROM History WHERE transcriptEntityId='one'").Scan(&text); err != nil || text != "Launch draft." {
		t.Fatalf("source changed: %q %v", text, err)
	}
}

func TestMissingWisprSourceDoesNotCreateDatabase(t *testing.T) {
	db, err := storage.NewDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := importWisprHistory(db, filepath.Join(t.TempDir(), "missing.sqlite")); err == nil {
		t.Fatal("missing source should fail")
	}
}
