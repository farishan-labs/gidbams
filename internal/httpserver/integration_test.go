package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPEndpoints(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "data", "items"), 0o755)
	os.WriteFile(filepath.Join(dir, "data", "items", "hello.md"), []byte("---\nid: hello\nname: Hello\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "gitdb.yaml"), []byte("version: 1\ndata_dir: data\ncollections:\n  - name: items\n    path: items\n"), 0o644)

	s := New(dir, "tok")
	h := s.Handler()
	for _, path := range []string{"/stats", "/records?collection=items", "/search?q=hello", "/validate"} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer tok")
		h.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Fatalf("%s status %d: %s", path, rr.Code, rr.Body.String())
		}
	}
}

