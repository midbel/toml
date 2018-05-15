package scan

import (
  "fmt"
  "strings"
)

func ExampleScanner_Multiline() {
  line := `
    description="""
the quick brown fox
jumps over
the lazy dog.
"""
  `
  s := NewScanner(strings.NewReader(line))
  t := s.Scan()
  fmt.Println(TokenString(t))
  t = s.Scan()
  fmt.Println(TokenString(t))

  t = s.Scan()
  fmt.Println(TokenString(t))
  r := strings.Map(func(r rune) rune{
    if r == '\n' || r == '\t' {
      return ' '
    }
    return r
  }, s.Text())
  fmt.Println(r)
  // Output:
  // ident
  // '='
  // string
  // """ the quick brown fox jumps over the lazy dog. """
}
