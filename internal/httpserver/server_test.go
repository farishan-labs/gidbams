package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHealth(t *testing.T) {
	s := New(t.TempDir(), "")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestStatsUnauthorized(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "data", "items"), 0o755)
	os.WriteFile(filepath.Join(dir, "gitdb.yaml"), []byte("version: 1\ndata_dir: data\ncollections:\n  - name: items\n    path: items\n"), 0o644)
	s := New(dir, "secret")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("status %d", rr.Code)
	}
}

