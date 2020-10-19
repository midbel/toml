package main

import (
	"flag"
	"fmt"
	"os"
	"io/ioutil"
	"path/filepath"

	"github.com/midbel/toml"
)

func main() {
	// overwrite := flag.Bool("w", false, "overwrite existing file")
	flag.Parse()
	for _, a := range flag.Args() {
		if err := writeDocument(a); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", a, err)
		}
	}
}

func overwriteDocument(doc string) error {
	r, err := os.Open(doc)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := ioutil.TempFile("", filepath.Base(doc))
	if err != nil {
		return err
	}
	defer w.Close()

	if err := toml.Format(r, w); err != nil {
		return err
	}
	return nil
}

func writeDocument(doc string) error {
	r, err := os.Open(doc)
	if err != nil {
		return err
	}
	defer r.Close()
	return toml.Format(r, os.Stdout)
}
