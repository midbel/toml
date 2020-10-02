package main

import (
  "flag"
  "fmt"
  "os"

  "github.com/midbel/toml"
)

func main() {
  flag.Parse()

  doc := make(map[string]interface{})
  if err := toml.DecodeFile(flag.Arg(0), &doc); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
  fmt.Println(doc)
}
