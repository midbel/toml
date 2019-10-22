package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/toml"
)

func main() {
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
