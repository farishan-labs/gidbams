package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/farishan-labs/git-dbms/internal/config"
	"github.com/farishan-labs/git-dbms/internal/store"
)

func TestBuild(t *testing.T) {
	dir := t.TempDir()
	def := &config.CollectionDefinition{
		Name:       "items",
		DataDir:    filepath.Join(dir, "items"),
		RecordFile: config.RecordFile{Format: "json", Name: "{id}.json"},
		Columns: []config.Column{
			{Name: "id", Type: "string", Required: true},
			{Name: "n", Type: "int"},
		},
	}
	os.MkdirAll(def.DataDir, 0o755)
	s := store.New(dir, def)
	if _, err := s.Create("a", map[string]any{"id": "a", "n": 5}, ""); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "out.db")
	b := New(dir, dbPath)
	err := b.Build([]struct {
		Def   *config.CollectionDefinition
		Store *store.Store
	}{{Def: def, Store: s}})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}
}
