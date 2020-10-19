package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/toml"
)

func main() {
	var (
		// overwrite = flag.Bool("w", false, "overwrite document")
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
	}
	for _, a := range flag.Args() {
		ft, err := toml.NewFormatter(a, rules...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if err := ft.Format(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
