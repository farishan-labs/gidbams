package store

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/farishan-labs/git-dbms/internal/config"
)

func TestConcurrentCreates(t *testing.T) {
	dir := t.TempDir()
	def := &config.CollectionDefinition{
		Name:       "items",
		DataDir:    dir + "/items",
		RecordFile: config.RecordFile{Format: "json", Name: "{id}.json"},
		Columns: []config.Column{
			{Name: "id", Type: "string", Required: true},
		},
	}
	os.MkdirAll(def.DataDir, 0o755)
	s := New(dir, def)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.Create(fmt.Sprintf("id-%d", i), map[string]any{"id": fmt.Sprintf("id-%d", i)}, "")
			if err != nil {
				t.Errorf("create failed: %v", err)
			}
		}(i)
	}
	wg.Wait()
	recs, err := s.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 20 {
		t.Fatalf("expected 20 records, got %d", len(recs))
	}
}

