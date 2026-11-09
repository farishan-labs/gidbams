package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRootSynthesize(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "data", "foo"), 0o755)
	c, err := LoadRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.DataDir != "data" {
		t.Fatalf("data dir: %s", c.DataDir)
	}
	if len(c.Collections) != 1 || c.Collections[0].Name != "foo" {
		t.Fatalf("collections: %+v", c.Collections)
	}
}

func TestSaveLoadRoot(t *testing.T) {
	dir := t.TempDir()
	c := &RootConfig{Version: 1, DataDir: "data", Settings: Settings{DefaultBranch: "main"}}
	if err := SaveRoot(dir, c); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 1 || got.DataDir != "data" {
		t.Fatalf("mismatch: %+v", got)
	}
}

