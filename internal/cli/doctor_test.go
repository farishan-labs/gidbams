package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDoctorCmdRuns(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "data", "items"), 0o755)
	os.WriteFile(filepath.Join(dir, "gitdb.yaml"), []byte("version: 1\ndata_dir: data\ncollections:\n  - name: items\n    path: items\n"), 0o644)
	repoRoot = dir
	if err := doctorCmd.RunE(doctorCmd, nil); err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
}

