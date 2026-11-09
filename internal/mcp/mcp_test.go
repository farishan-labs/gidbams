package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestInitialize(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n")
	var out bytes.Buffer
	s.Stdin = in
	s.Stdout = &out
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "git-dbms") {
		t.Fatalf("expected server info, got %s", out.String())
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		lines = append(lines, l)
	}
	// parse second response
	if len(lines) < 2 {
		t.Fatalf("expected 2 responses, got %d", len(lines))
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &resp); err != nil {
		t.Fatal(err)
	}
	res := resp["result"].(map[string]any)
	tools := res["tools"].([]any)
	if len(tools) < 5 {
		t.Fatalf("expected tools, got %d", len(tools))
	}
}

