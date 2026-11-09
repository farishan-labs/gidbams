package mcp

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResources(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "data", "items"), 0o755)
	os.WriteFile(filepath.Join(dir, "data", "items", "hello.md"), []byte("---\nid: hello\nname: Hello\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "gitdb.yaml"), []byte("version: 1\ndata_dir: data\ncollections:\n  - name: items\n    path: items\n"), 0o644)

	srv := New(dir)
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"resources/list","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"gitdb://items/hello"}}` + "\n")
	var out bytes.Buffer
	srv.Stdin = in
	srv.Stdout = &out
	if err := srv.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("expected hello: %s", out.String())
	}
}
