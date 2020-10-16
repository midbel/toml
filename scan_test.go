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
		{Literal: "comment", Type: TokComment},
		{Literal: "bool", Type: TokString},
		{Type: TokEqual},
		{Literal: "true", Type: TokBool},
		{Literal: "bool", Type: TokString},
		{Type: TokEqual},
		{Literal: "false", Type: TokBool},
		{Type: TokBegRegularTable},
		{Literal: "number", Type: TokIdent},
		{Type: TokEndRegularTable},
		{Literal: "integer", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "100", Type: TokInteger},
		{Literal: "positive", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "100", Type: TokInteger},
		{Literal: "negative", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "-100", Type: TokInteger},
		{Literal: "underscore", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "123_456", Type: TokInteger},
		{Literal: "hexa", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "0xdead_beef", Type: TokInteger},
		{Literal: "octal", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "0o1234567", Type: TokInteger},
		{Literal: "binary", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "0b0000001", Type: TokInteger},
		{Literal: "zero", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "0", Type: TokInteger},
		{Literal: "float1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "3.14", Type: TokFloat},
		{Literal: "float2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "1e16", Type: TokFloat},
		{Literal: "float3", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "3.14_15", Type: TokFloat},
		{Literal: "float4", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "-1e16_185", Type: TokFloat},
		{Literal: "float5", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "3.14E1_987", Type: TokFloat},
		{Type: TokBegRegularTable},
		{Literal: "strings", Type: TokIdent},
		{Type: TokEndRegularTable},
		{Literal: "empty1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "", Type: TokString},
		{Literal: "empty2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "", Type: TokString},
		{Literal: "basic", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "basic string", Type: TokString},
		{Literal: "literal", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "literal string", Type: TokString},
		{Literal: "multiline1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "The quick brown fox jumps over the lazy dog.", Type: TokString},
		{Literal: "multiline2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "Roses are red\nViolets are blue", Type: TokString},
		{Type: TokBegRegularTable},
		{Literal: "date", Type: TokIdent},
		{Type: TokEndRegularTable},
		{Literal: "odt1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "2019-10-24T19:07:54Z", Type: TokDatetime},
		{Literal: "odt2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "2019-10-24 19:07:54+02:00", Type: TokDatetime},
		{Literal: "odt3", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "2019-10-24T19:07:54.123Z", Type: TokDatetime},
		{Literal: "date", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "2019-10-24", Type: TokDate},
		{Literal: "time1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "19:07:54", Type: TokTime},
		{Literal: "time2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "09:07:54", Type: TokTime},
		{Type: TokBegRegularTable},
		{Literal: "container", Type: TokIdent},
		{Type: TokEndRegularTable},
		{Literal: "array1", Type: TokIdent},
		{Type: TokEqual},
		{Type: TokBegArray},
		{Literal: "1", Type: TokInteger},
		{Type: TokComma},
		{Literal: "2", Type: TokInteger},
		{Type: TokComma},
		{Literal: "3", Type: TokInteger},
		{Type: TokComma},
		{Type: TokEndArray},
		{Literal: "array2", Type: TokIdent},
		{Type: TokEqual},
		{Type: TokBegArray},
		{Literal: "basic", Type: TokString},
		{Type: TokComma},
		{Literal: "literal", Type: TokString},
		{Type: TokEndArray},
		{Literal: "array3", Type: TokIdent},
		{Type: TokEqual},
		{Type: TokBegArray},
		{Type: TokBegArray},
		{Literal: "1", Type: TokInteger},
		{Type: TokComma},
		{Literal: "2", Type: TokInteger},
		{Type: TokComma},
		{Literal: "3", Type: TokInteger},
		{Type: TokComma},
		{Type: TokEndArray},
		{Type: TokComma},
		{Type: TokBegArray},
		{Literal: "foo", Type: TokString},
		{Type: TokComma},
		{Literal: "bar", Type: TokString},
		{Type: TokComma},
		{Type: TokEndArray},
		{Type: TokComma},
		{Type: TokEndArray},
		{Literal: "inline", Type: TokIdent},
		{Type: TokEqual},
		{Type: TokBegInline},
		{Literal: "key", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "foo", Type: TokString},
		{Type: TokComma},
		{Literal: "active", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "true", Type: TokBool},
		{Type: TokComma},
		{Literal: "number", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "100", Type: TokInteger},
		{Type: TokEndInline},
	}

	s, err := NewScanner(strings.NewReader(strings.TrimSpace(doc)))
	if err != nil {
		t.Fatalf("fail to prepare scanner: %s", err)
	}

	var i int
	for got := s.Scan(); got.Type != TokEOF; got = s.Scan() {
		if got.Type == TokIllegal {
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
	if got.Type != TokEOF {
		t.Fatalf("last token is not EOF")
	}
}
