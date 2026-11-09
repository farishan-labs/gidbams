package record

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTripMarkdown(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.md")
	r := &Record{Path: p, Format: "md", Fields: map[string]any{"id": "x", "name": "X"}, Body: "hello"}
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "x" || got.Fields["name"] != "X" || got.Body != "hello\n" {
		t.Fatalf("mismatch: %+v", got)
	}
	_ = os.Remove(p)
}

