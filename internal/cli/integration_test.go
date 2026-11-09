package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLIIntegration(t *testing.T) {
	dir := t.TempDir()
	old := repoRoot
	repoRoot = dir
	defer func() { repoRoot = old }()

	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatal(err)
	}
	// create collection via collection create
	if err := collectionCreateCmd.RunE(collectionCreateCmd, []string{"items"}); err != nil {
		t.Fatal(err)
	}
	// create record file manually
	recPath := filepath.Join(dir, "data", "items", "hello.md")
	if err := os.WriteFile(recPath, []byte("---\nid: hello\nname: Hello\n---\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateCmd.RunE(validateCmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := buildCmd.RunE(buildCmd, nil); err != nil {
		t.Fatal(err)
	}
	// query
	// no direct output capture needed; just ensure no error
	if err := queryCmd.RunE(queryCmd, []string{"SELECT COUNT(*) FROM items"}); err != nil {
		t.Fatal(err)
	}
	if err := statsCmd.RunE(statsCmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := doctorCmd.RunE(doctorCmd, nil); err != nil {
		t.Fatal(err)
	}
}

