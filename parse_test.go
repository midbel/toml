package toml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	files := []struct {
		File  string
		Valid bool
	}{
		{File: "numbers.toml", Valid: true},
		{File: "strings.toml", Valid: true},
		{File: "booleans.toml", Valid: true},
		{File: "dates.toml", Valid: true},
		{File: "arrays.toml", Valid: true},
		{File: "inlines.toml", Valid: true},
		{File: "package.toml", Valid: true},
	}
	for _, f := range files {
		r, err := os.Open(filepath.Join("testdata", f.File))
		if err != nil {
			t.Error(err)
		}
		if _, err := Parse(r); err != nil && f.Valid {
			t.Errorf("%s: unexpected error when parsing: %s", f.File, err)
		}
	}
}
