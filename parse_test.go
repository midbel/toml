package toml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	files := []string{
		"numbers",
		"strings",
		"booleans",
		"arrays",
		"array1.bad",
		"array2.bad",
		"inlines",
		"inline1.bad",
		"inline2.bad",
		"inline3.bad",
		"keys",
		"key1.bad",
		"key2.bad",
		"key3.bad",
		"key4.bad",
		"key5.bad",
		"tables",
		"table1.bad",
		"table2.bad",
		"table3.bad",
		"table4.bad",
		"table5.bad",
		"table6.bad",
		"package",
		"fruits",
		"example",
	}
	for _, f := range files {
		file := f + ".toml"

		r, err := os.Open(filepath.Join("testdata", file))
		if err != nil {
			t.Error(err)
			continue
		}

		valid := filepath.Ext(f) == ""
		switch _, err := Parse(r); {
		case valid && err != nil:
			t.Errorf("%s: %s", file, err)
		case !valid && err == nil:
			t.Errorf("%s: invalid document not detected", file)
		}
		r.Close()
	}
}
