package toml

import (
	"strings"
	"testing"
)

func TestScannerScan(t *testing.T) {
	doc := `
# comment

"bool" = true
"bool" = false

[number]
integer    = 100
positive   = +100
negative   = -100
underscore = 123_456
hexa       = 0xdead_beef
octal      = 0o1234567
binary     = 0b0000001
zero       = 0
float1     = 3.14
float2     = 1e16
float3     = 3.14_15
float4     = -1e16_185
float5     = +3.14E1_987


[strings]
empty1  = ""
empty2  = ''
basic   = "basic string"
literal = "literal string"

multiline1 = """\
       The quick brown \
       fox jumps over \
       the lazy dog.\
"""
multiline2 = """
Roses are red
Violets are blue"""

[date]
odt1 = 2019-10-24T19:07:54Z
odt2 = 2019-10-24 19:07:54+02:00
odt3 = 2019-10-24T19:07:54.123Z
date = 2019-10-24
time1 = 19:07:54
time2 = 09:07:54

[container]
array1 = [1, 2, 3, ]
array2 = ["basic", 'literal']
array3 = [
	[1, 2, 3, ],
	["foo", "bar", ],
]

inline = {key = "foo", active = true, number = 100}
`

	tokens := []Token{
		{Literal: "comment", Type: Comment},
		{Literal: "bool", Type: String},
		{Type: equal},
		{Literal: "true", Type: Bool},
		{Type: Newline},
		{Literal: "bool", Type: String},
		{Type: equal},
		{Literal: "false", Type: Bool},
		{Type: Newline},
		{Type: lsquare},
		{Literal: "number", Type: Ident},
		{Type: rsquare},
		{Type: Newline},
		{Literal: "integer", Type: Ident},
		{Type: equal},
		{Literal: "100", Type: Integer},
		{Type: Newline},
		{Literal: "positive", Type: Ident},
		{Type: equal},
		{Literal: "100", Type: Integer},
		{Type: Newline},
		{Literal: "negative", Type: Ident},
		{Type: equal},
		{Literal: "-100", Type: Integer},
		{Type: Newline},
		{Literal: "underscore", Type: Ident},
		{Type: equal},
		{Literal: "123_456", Type: Integer},
		{Type: Newline},
		{Literal: "hexa", Type: Ident},
		{Type: equal},
		{Literal: "0xdead_beef", Type: Integer},
		{Type: Newline},
		{Literal: "octal", Type: Ident},
		{Type: equal},
		{Literal: "0o1234567", Type: Integer},
		{Type: Newline},
		{Literal: "binary", Type: Ident},
		{Type: equal},
		{Literal: "0b0000001", Type: Integer},
		{Type: Newline},
		{Literal: "zero", Type: Ident},
		{Type: equal},
		{Literal: "0", Type: Integer},
		{Type: Newline},
		{Literal: "float1", Type: Ident},
		{Type: equal},
		{Literal: "3.14", Type: Float},
		{Type: Newline},
		{Literal: "float2", Type: Ident},
		{Type: equal},
		{Literal: "1e16", Type: Float},
		{Type: Newline},
		{Literal: "float3", Type: Ident},
		{Type: equal},
		{Literal: "3.14_15", Type: Float},
		{Type: Newline},
		{Literal: "float4", Type: Ident},
		{Type: equal},
		{Literal: "-1e16_185", Type: Float},
		{Type: Newline},
		{Literal: "float5", Type: Ident},
		{Type: equal},
		{Literal: "3.14E1_987", Type: Float},
		{Type: Newline},
		{Type: lsquare},
		{Literal: "strings", Type: Ident},
		{Type: rsquare},
		{Type: Newline},
		{Literal: "empty1", Type: Ident},
		{Type: equal},
		{Literal: "", Type: String},
		{Type: Newline},
		{Literal: "empty2", Type: Ident},
		{Type: equal},
		{Literal: "", Type: String},
		{Type: Newline},
		{Literal: "basic", Type: Ident},
		{Type: equal},
		{Literal: "basic string", Type: String},
		{Type: Newline},
		{Literal: "literal", Type: Ident},
		{Type: equal},
		{Literal: "literal string", Type: String},
		{Type: Newline},
		{Literal: "multiline1", Type: Ident},
		{Type: equal},
		{Literal: "The quick brown fox jumps over the lazy dog.", Type: String},
		{Type: Newline},
		{Literal: "multiline2", Type: Ident},
		{Type: equal},
		{Literal: "Roses are red\nViolets are blue", Type: String},
		{Type: Newline},
		{Type: lsquare},
		{Literal: "date", Type: Ident},
		{Type: rsquare},
		{Type: Newline},
		{Literal: "odt1", Type: Ident},
		{Type: equal},
		{Literal: "2019-10-24T19:07:54Z", Type: DateTime},
		{Type: Newline},
		{Literal: "odt2", Type: Ident},
		{Type: equal},
		{Literal: "2019-10-24 19:07:54+02:00", Type: DateTime},
		{Type: Newline},
		{Literal: "odt3", Type: Ident},
		{Type: equal},
		{Literal: "2019-10-24T19:07:54.123Z", Type: DateTime},
		{Type: Newline},
		{Literal: "date", Type: Ident},
		{Type: equal},
		{Literal: "2019-10-24", Type: Date},
		{Type: Newline},
		{Literal: "time1", Type: Ident},
		{Type: equal},
		{Literal: "19:07:54", Type: Time},
		{Type: Newline},
		{Literal: "time2", Type: Ident},
		{Type: equal},
		{Literal: "09:07:54", Type: Time},
		{Type: Newline},
		{Type: lsquare},
		{Literal: "container", Type: Ident},
		{Type: rsquare},
		{Type: Newline},
		{Literal: "array1", Type: Ident},
		{Type: equal},
		{Type: lsquare},
		{Literal: "1", Type: Integer},
		{Type: comma},
		{Literal: "2", Type: Integer},
		{Type: comma},
		{Literal: "3", Type: Integer},
		{Type: comma},
		{Type: rsquare},
		{Type: Newline},
		{Literal: "array2", Type: Ident},
		{Type: equal},
		{Type: lsquare},
		{Literal: "basic", Type: String},
		{Type: comma},
		{Literal: "literal", Type: String},
		{Type: rsquare},
		{Type: Newline},
		{Literal: "array3", Type: Ident},
		{Type: equal},
		{Type: lsquare},
		{Type: Newline},
		{Type: lsquare},
		{Literal: "1", Type: Integer},
		{Type: comma},
		{Literal: "2", Type: Integer},
		{Type: comma},
		{Literal: "3", Type: Integer},
		{Type: comma},
		{Type: rsquare},
		{Type: comma},
		{Type: Newline},
		{Type: lsquare},
		{Literal: "foo", Type: String},
		{Type: comma},
		{Literal: "bar", Type: String},
		{Type: comma},
		{Type: rsquare},
		{Type: comma},
		{Type: Newline},
		{Type: rsquare},
		{Type: Newline},
		{Literal: "inline", Type: Ident},
		{Type: equal},
		{Type: lcurly},
		{Literal: "key", Type: Ident},
		{Type: equal},
		{Literal: "foo", Type: String},
		{Type: comma},
		{Literal: "active", Type: Ident},
		{Type: equal},
		{Literal: "true", Type: Bool},
		{Type: comma},
		{Literal: "number", Type: Ident},
		{Type: equal},
		{Literal: "100", Type: Integer},
		{Type: rcurly},
		{Type: Newline},
	}

	s, err := Scan(strings.NewReader(strings.TrimSpace(doc)))
	if err != nil {
		t.Fatalf("fail to prepare scanner: %s", err)
	}
	s.KeepComment = true
	s.KeepMultiline = false

	var i int
	for got := s.Scan(); got.Type != EOF; got = s.Scan() {
		if got.Type == Illegal {
			t.Fatalf("<%s> illegal token found: %s", got.Pos, got)
		}
		want := tokens[i]
		if want.Type != got.Type || want.Literal != got.Literal {
			t.Fatalf("<%s> unexpected token: want %s, got %s", got.Pos, want, got)
		}
		i++
		if i >= len(tokens) {
			t.Fatalf("too many tokens")
			break
		}
	}
	got := s.Scan()
	if got.Type != EOF {
		t.Fatalf("last token is not EOF")
	}
}
