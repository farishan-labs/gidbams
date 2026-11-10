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

func TestLoadRootRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "gitdb.yaml"), []byte("version: 1\ndata_dir: data\ncollections:\n  - name: bad\n    path: ../etc\n"), 0o644)
	if _, err := LoadRoot(dir); err == nil {
		t.Fatal("expected error")
	}
}

func TestEnvOrPrecedence(t *testing.T) {
	t.Setenv(EnvDBNew, "new-val")
	t.Setenv(EnvDBLegacy, "legacy-val")
	if got := EnvOr(EnvDBNew, EnvDBLegacy); got != "new-val" {
		t.Fatalf("precedence: got %q want new-val", got)
	}
}

func TestEnvOrFallback(t *testing.T) {
	t.Setenv(EnvDBNew, "")
	t.Setenv(EnvDBLegacy, "legacy-val")
	if got := EnvOr(EnvDBNew, EnvDBLegacy); got != "legacy-val" {
		t.Fatalf("fallback: got %q want legacy-val", got)
	}
}

func TestEnvOrEmpty(t *testing.T) {
	t.Setenv(EnvDBNew, "")
	t.Setenv(EnvDBLegacy, "")
	if got := EnvOr(EnvDBNew, EnvDBLegacy); got != "" {
		t.Fatalf("empty: got %q want empty", got)
	}
}

func TestEnvDB(t *testing.T) {
	t.Setenv(EnvDBNew, "newdb")
	t.Setenv(EnvDBLegacy, "legacydb")
	if got := EnvDB(); got != "newdb" {
		t.Fatalf("EnvDB precedence: got %q", got)
	}
	t.Setenv(EnvDBNew, "")
	if got := EnvDB(); got != "legacydb" {
		t.Fatalf("EnvDB fallback: got %q", got)
	}
}

func TestEnvToken(t *testing.T) {
	t.Setenv(EnvTokenNew, "newtok")
	t.Setenv(EnvTokenLegacy, "legacytok")
	if got := EnvToken(); got != "newtok" {
		t.Fatalf("EnvToken precedence: got %q", got)
	}
	t.Setenv(EnvTokenNew, "")
	if got := EnvToken(); got != "legacytok" {
		t.Fatalf("EnvToken fallback: got %q", got)
	}
}
