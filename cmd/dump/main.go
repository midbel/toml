package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/toml"
)

var help = `tomldump write the AST of a TOML document to stdout

usage: tomldump <document.toml>`

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stdout, help)
		os.Exit(1)
	}
	flag.Parse()

	for i := 0; i < flag.NArg(); i++ {
		err := dumpFile(flag.Arg(i))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func dumpFile(file string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()

	n, err := toml.Parse(r)
	if err == nil {
		toml.Dump(n)
	}
	return err

}
