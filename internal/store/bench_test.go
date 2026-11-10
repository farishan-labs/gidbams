package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/farishan-labs/gidbams/internal/config"
)

func BenchmarkLoadAll(b *testing.B) {
	dir := b.TempDir()
	def := &config.CollectionDefinition{
		Name:       "items",
		DataDir:    filepath.Join(dir, "items"),
		RecordFile: config.RecordFile{Format: "json", Name: "{id}.json"},
		Columns: []config.Column{
			{Name: "id", Type: "string", Required: true},
		},
	}
	os.MkdirAll(def.DataDir, 0o755)
	s := New(dir, def)
	for i := 0; i < 100; i++ {
		_, _ = s.Create(fmt.Sprintf("id-%d", i), map[string]any{"id": fmt.Sprintf("id-%d", i)}, "")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.LoadAll()
	}
}
