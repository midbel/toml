package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/toml"
)

func main() {
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	sc, err := toml.Scan(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	for k := sc.Scan(); k.Type != toml.EOF; k = sc.Scan() {
		fmt.Println(k)
	}
}
