package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farishan-labs/git-dbms/internal/config"
)

func TestStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	def := &config.CollectionDefinition{
		Name:       "items",
		DataDir:    filepath.Join(dir, "items"),
		RecordFile: config.RecordFile{Format: "md", Name: "{id}.md"},
	}
	os.MkdirAll(def.DataDir, 0o755)
	s := New(dir, def)
	r, err := s.Create("a", map[string]any{"id": "a", "name": "A"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "a" {
		t.Fatalf("id mismatch")
	}
	got, err := s.Get("a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Fields["name"] != "A" {
		t.Fatalf("name mismatch")
	}
	if _, err := s.Create("a", map[string]any{"id": "a"}, ""); err == nil {
		t.Fatal("expected duplicate error")
	}
	if err := s.Delete("a"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("a"); err == nil {
		t.Fatal("expected not found")
	}
}

