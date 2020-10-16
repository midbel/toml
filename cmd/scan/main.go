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

	s, err := toml.NewScanner(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	for k := s.Scan(); k.Type != toml.TokEOF; k = s.Scan() {
		fmt.Println(k)
	}
}
