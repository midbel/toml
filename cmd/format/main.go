package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/toml"
)

const help = `tomlfmt re writes a toml document.

options:

  -a  FMT   rewrite array(s) according to FMT
  -d  FMT   use FMT as base when rewritting integers
  -e  EOL   use EOL when writing the end of line
  -f  FMT   use FMT to rewrite floats
  -g        use UTC time for datetime values
  -h        print this help message and exit
  -i        rewrite (array of) inline table(s) to (array of) regular table(s)
  -k        keep empty table(s) when rewritting document
  -m  PREC  use PREC as millisecond precision for datetime values
  -n        nest sub tables with indentation
  -o        remove comments from document
  -r        keep raw values
  -s  SPACE use SPACE space(s) as indent instead of tab
  -u  NUM   insert underscore in number (integer/float) every NUM characters
  -w        overwrite source file

Array format:

* multi: force arrays to be written on multiple lines (except for arrays with
less that 2 elements)
* single: force arrays to be written on a single line
* mixed (default): arrays will be written as they are in the original document.

Integer base:

* x, hex: all integers will be written in hexadecimal
* o, oct: all integers will be written in octal
* b, bin: all integers will be written in binary
* d, dec (default): all integers will be written in decimal

Float format:

* f: floats will be written in normal notation without exponent
* e: floats will be written in scientific notation
* g: floats will be written, depending of their values, to normal or scientific notation

End of Line:

* lf: use line feed as end of line terminator
* crlf: use carriage return, line feed as end of line terminator

`

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stdout, help)
		os.Exit(2)
	}
	var (
		overwrite = flag.Bool("w", false, "overwrite document")
		// general option
		raw   = flag.Bool("r", false, "keep raw values")
		keep  = flag.Bool("k", false, "keep empty table(s)")
		nest  = flag.Bool("n", false, "nest sub table(s)")
		space = flag.Int("s", 0, "use space for indentation instead of tab")
		nocom = flag.Bool("o", false, "ignore comment(s)")
		eol   = flag.String("e", "", "end of line")
		// time formatting options
		utc    = flag.Bool("g", false, "convert local date time to UTC date time")
		millis = flag.Int("m", 0, "use given millis precision")
		// number formatting options
		float      = flag.String("f", "", "format float with the given base")
		decimal    = flag.String("d", "", "format integer with the given base")
		underscore = flag.Int("u", 0, "insert underscore in number (float/integer)")
		// array/inline formatting option
		array  = flag.String("a", "", "write array on multiple/single line(s)")
		inline = flag.Bool("i", false, "convert inline table(s) to regular table(s)")
	)
	flag.Parse()
	rules := []toml.FormatRule{
		toml.WithTab(*space),
		toml.WithEmpty(*keep),
		toml.WithNest(*nest),
		toml.WithFloat(*float, *underscore),
		toml.WithNumber(*decimal, *underscore),
		toml.WithComment(!*nocom),
		toml.WithTime(*millis, *utc),
		toml.WithArray(*array),
		toml.WithInline(*inline),
		toml.WithEOL(*eol),
		toml.WithRaw(*raw),
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
