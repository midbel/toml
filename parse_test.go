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
		{File: "inline2.bad.toml", Valid: false},
		{File: "package.toml", Valid: true},
	}
	for _, f := range files {
		r, err := os.Open(filepath.Join("testdata", f.File))
		if err != nil {
			t.Error(err)
			continue
		}

		switch _, err := Parse(r); {
		case f.Valid && err != nil:
			t.Errorf("%s: %s", f.File, err)
		case !f.Valid && err == nil:
			t.Errorf("%s: invalid document not detected", f.File)
		}
		r.Close()
	}
}
