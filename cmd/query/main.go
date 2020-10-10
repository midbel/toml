package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/midbel/query"
	"github.com/midbel/toml"
)

const (
	ExitBadQuery int = iota + 1
	ExitBadDoc
	ExitEmpty
)

func main() {
	path := flag.Bool("p", false, "print path")
	flag.Parse()

	q, err := query.Parse(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, flag.Arg(0), err)
		os.Exit(ExitBadQuery)
	}

	doc := make(map[string]interface{})
	if err := toml.DecodeFile(flag.Arg(1), &doc); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitBadDoc)
	}
	ifi, err := q.Select(doc)
	switch {
	case err != nil:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitBadQuery)
	case ifi == nil:
		os.Exit(ExitEmpty)
	default:
	}
	var (
		root  = filepath.Base(flag.Arg(1))
		print = nokey
	)
	if *path {
		print = withkey
	}
	printResults(strings.TrimSuffix(root, ".toml"), ifi, print)
}

func nokey(_ string, value interface{}) {
	fmt.Println(value)
}

func withkey(key string, value interface{}) {
	fmt.Printf("%s = %v\n", key, value)
}

func printResults(key string, value interface{}, print func(string, interface{})) {
	switch ifi := value.(type) {
	case []interface{}:
		if len(ifi) == 1 {
			printResults(key, ifi[0], print)
			return
		}
		for j, i := range ifi {
			printResults(fmt.Sprintf("%s.%d", key, j), i, print)
		}
	case map[string]interface{}:
		for k, v := range ifi {
			printResults(fmt.Sprintf("%s.%s", key, k), v, print)
		}
	default:
		print(key, ifi)
	}
}
