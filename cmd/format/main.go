package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/toml"
)

func main() {
	flag.Parse()
	for _, a := range flag.Args() {
		if err := writeDocument(a); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", a, err)
		}
	}
}

func writeDocument(doc string) error {
	r, err := os.Open(doc)
	if err != nil {
		return err
	}
	defer r.Close()
	return toml.Format(r, os.Stdout)
}
