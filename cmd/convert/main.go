package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/midbel/toml"
)

func main() {
	flag.Parse()

	w, err := os.Create(getFile(flag.Arg(0)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer w.Close()

	if err := save(w, flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func save(w io.Writer, file string) error {
	var in interface{}
	if err := toml.DecodeFile(file, &in); err != nil {
		return err
	}
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(in)
}

func getFile(file string) string {
	file = strings.TrimRight(file, filepath.Ext(file))
	return file + ".json"
}
