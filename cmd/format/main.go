package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/toml"
)

func main() {
	var (
		overwrite = flag.Bool("w", false, "overwrite document")
		// general option
		keep  = flag.Bool("k", false, "keep empty table(s)")
		nest  = flag.Bool("n", false, "nest sub table(s)")
		space = flag.Int("s", 0, "use space for indentation instead of tab")
		// time formatting options
		// utc    = flag.Bool("u", false, "convert local date time to UTC date time")
		// millis = flag.Int("m", 0, "use given millis precision")
		// number formatting options
		float   = flag.String("f", "", "format float with the given base")
		decimal = flag.String("d", "", "format integer with the given base")
	)
	flag.Parse()
	rules := []toml.FormatRule{
		toml.WithTab(*space),
		toml.WithEmpty(*keep),
		toml.WithNest(*nest),
		toml.WithFloat(*float),
		toml.WithNumber(*decimal),
		toml.WithComment(true),
	}
	for _, a := range flag.Args() {
		if err := formatDocument(a, *overwrite, rules); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func formatDocument(doc string, overwrite bool, rules []toml.FormatRule) error {
	ft, err := toml.NewFormatter(doc, rules...)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := ft.Format(&buf); err != nil {
		return err
	}
	out := os.Stdout
	if overwrite {
		w, err := os.Create(doc)
		if err != nil {
			return err
		}
		defer w.Close()
		out = w
	}
	_, err = io.Copy(out, &buf)
	return err
}
