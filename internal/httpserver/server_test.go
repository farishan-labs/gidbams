package httpserver

import (
	"net/http"
	"net/http/httptest"
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

