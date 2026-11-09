package record

import (
	"testing"
)

func FuzzParseFrontmatter(f *testing.F) {
	f.Add([]byte("---\nid: x\nname: X\n---\nbody"))
	f.Add([]byte("no frontmatter"))
	f.Add([]byte("---\nbad: ["))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = parseFrontmatter(data)
	})
}

