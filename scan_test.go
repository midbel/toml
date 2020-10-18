package toml

import (
	"strings"
	"testing"
)

func TestScannerScan(t *testing.T) {
	doc := `
# a comment #1

"bool1" = true
"bool2" = false
3.14    = "pi"

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

"inf"  = inf
"+inf" = +inf
"-inf" = -inf
"nan"  = nan
"+nan" = +nan
"-nan" = -nan


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

quote.single = '''quote can be 'quoted'!'''
quote.double = """quote can be ""quoted""!"""

unicode1 = "a heart \u2665!"
unicode2 = "not a heart \\u2665!"

[date]
odt1 = 2019-10-24T19:07:54Z
odt2 = 2019-10-24 19:07:54+02:00
odt3 = 2019-10-24T19:07:54.123Z
date = 2019-10-24
time1 = 19:07:54
time2 = 09:07:54

[[container]]
array1 = [1, 2, 3, ]
array2 = ["basic", 'literal']
array3 = [
	[1, 2, 3, ],
	["foo", "bar", ],
	{1234 = "number", bool = true, array = [1, 2, 3], "roles" = [
		"user",
		"admin",
		"guest", # a comment
	]},
]

inline = {key = "foo", active = true, number = 100}

[illegal]
key = "value" illegal = 1234
`
	nl := Token{Type: TokNL}
	tokens := []Token{
		{Literal: "a comment #1", Type: TokComment},
		nl,
		{Literal: "bool1", Type: TokString},
		{Type: TokEqual},
		{Literal: "true", Type: TokBool},
		nl,
		{Literal: "bool2", Type: TokString},
		{Type: TokEqual},
		{Literal: "false", Type: TokBool},
		nl,
		{Literal: "3", Type: TokIdent},
		{Type: TokDot},
		{Literal: "14", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "pi", Type: TokString},
		nl,
		{Type: TokBegRegularTable},
		{Literal: "number", Type: TokIdent},
		{Type: TokEndRegularTable},
		nl,
		{Literal: "integer", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "100", Type: TokInteger},
		nl,
		{Literal: "positive", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "100", Type: TokInteger},
		nl,
		{Literal: "negative", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "-100", Type: TokInteger},
		nl,
		{Literal: "underscore", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "123456", Type: TokInteger},
		nl,
		{Literal: "hexa", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "0xdeadbeef", Type: TokInteger},
		nl,
		{Literal: "octal", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "0o1234567", Type: TokInteger},
		nl,
		{Literal: "binary", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "0b0000001", Type: TokInteger},
		nl,
		{Literal: "zero", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "0", Type: TokInteger},
		nl,
		{Literal: "float1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "3.14", Type: TokFloat},
		nl,
		{Literal: "float2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "1e16", Type: TokFloat},
		nl,
		{Literal: "float3", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "3.1415", Type: TokFloat},
		nl,
		{Literal: "float4", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "-1e16185", Type: TokFloat},
		nl,
		{Literal: "float5", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "3.14E1987", Type: TokFloat},
		nl,
		{Literal: "inf", Type: TokString},
		{Type: TokEqual},
		{Literal: "inf", Type: TokFloat},
		nl,
		{Literal: "+inf", Type: TokString},
		{Type: TokEqual},
		{Literal: "+inf", Type: TokFloat},
		nl,
		{Literal: "-inf", Type: TokString},
		{Type: TokEqual},
		{Literal: "-inf", Type: TokFloat},
		nl,
		{Literal: "nan", Type: TokString},
		{Type: TokEqual},
		{Literal: "nan", Type: TokFloat},
		nl,
		{Literal: "+nan", Type: TokString},
		{Type: TokEqual},
		{Literal: "+nan", Type: TokFloat},
		nl,
		{Literal: "-nan", Type: TokString},
		{Type: TokEqual},
		{Literal: "-nan", Type: TokFloat},
		nl,
		{Type: TokBegRegularTable},
		{Literal: "strings", Type: TokIdent},
		{Type: TokEndRegularTable},
		nl,
		{Literal: "empty1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "", Type: TokString},
		nl,
		{Literal: "empty2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "", Type: TokString},
		nl,
		{Literal: "basic", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "basic string", Type: TokString},
		nl,
		{Literal: "literal", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "literal string", Type: TokString},
		nl,
		{Literal: "multiline1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "The quick brown fox jumps over the lazy dog.", Type: TokString},
		nl,
		{Literal: "multiline2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "Roses are red\nViolets are blue", Type: TokString},
		nl,
		{Literal: "quote", Type: TokIdent},
		{Type: TokDot},
		{Literal: "single", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "quote can be 'quoted'!", Type: TokString},
		nl,
		{Literal: "quote", Type: TokIdent},
		{Type: TokDot},
		{Literal: "double", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "quote can be \"\"quoted\"\"!", Type: TokString},
		nl,
		{Literal: "unicode1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "a heart \u2665!", Type: TokString},
		nl,
		{Literal: "unicode2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "not a heart \\u2665!", Type: TokString},
		nl,
		{Type: TokBegRegularTable},
		{Literal: "date", Type: TokIdent},
		{Type: TokEndRegularTable},
		nl,
		{Literal: "odt1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "2019-10-24T19:07:54Z", Type: TokDatetime},
		nl,
		{Literal: "odt2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "2019-10-24 19:07:54+02:00", Type: TokDatetime},
		nl,
		{Literal: "odt3", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "2019-10-24T19:07:54.123Z", Type: TokDatetime},
		nl,
		{Literal: "date", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "2019-10-24", Type: TokDate},
		nl,
		{Literal: "time1", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "19:07:54", Type: TokTime},
		nl,
		{Literal: "time2", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "09:07:54", Type: TokTime},
		nl,
		{Type: TokBegArrayTable},
		{Literal: "container", Type: TokIdent},
		{Type: TokEndArrayTable},
		nl,
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
		nl,
		{Literal: "array2", Type: TokIdent},
		{Type: TokEqual},
		{Type: TokBegArray},
		{Literal: "basic", Type: TokString},
		{Type: TokComma},
		{Literal: "literal", Type: TokString},
		{Type: TokEndArray},
		nl,
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
		{Type: TokBegInline},
		{Literal: "1234", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "number", Type: TokString},
		{Type: TokComma},
		{Literal: "bool", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "true", Type: TokBool},
		{Type: TokComma},
		{Literal: "array", Type: TokIdent},
		{Type: TokEqual},
		{Type: TokBegArray},
		{Literal: "1", Type: TokInteger},
		{Type: TokComma},
		{Literal: "2", Type: TokInteger},
		{Type: TokComma},
		{Literal: "3", Type: TokInteger},
		{Type: TokEndArray},
		{Type: TokComma},
		{Literal: "roles", Type: TokString},
		{Type: TokEqual},
		{Type: TokBegArray},
		{Literal: "user", Type: TokString},
		{Type: TokComma},
		{Literal: "admin", Type: TokString},
		{Type: TokComma},
		{Literal: "guest", Type: TokString},
		{Type: TokComma},
		{Literal: "a comment", Type: TokComment},
		{Type: TokEndArray},
		{Type: TokEndInline},
		{Type: TokComma},
		{Type: TokEndArray},
		nl,
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
		nl,
		{Type: TokBegRegularTable},
		{Literal: "illegal", Type: TokIdent},
		{Type: TokEndRegularTable},
		nl,
		{Literal: "key", Type: TokIdent},
		{Type: TokEqual},
		{Literal: "value", Type: TokString},
		{Literal: "illegal = 1234", Type: TokIllegal},
	}

	s, err := NewScanner(strings.NewReader(strings.TrimSpace(doc)))
	if err != nil {
		t.Fatalf("fail to prepare scanner: %s", err)
	}

	var (
		i    int
		prev Token
	)
	for got := s.Scan(); got.Type != TokEOF; got = s.Scan() {
		if i >= len(tokens) {
			t.Fatalf("too many tokens! want %d, got %d", len(tokens), i)
			break
		}
		// if got.Type == TokIllegal {
		// 	t.Fatalf("<%s> illegal token found: %s", got.Pos, got)
		// }
		want := tokens[i]
		if want.Type != got.Type || want.Literal != got.Literal {
			t.Fatalf("<%s> unexpected token after %s: want %s, got %s", got.Pos, prev, want, got)
		}
		prev = got
		i++
	}
	got := s.Scan()
	if got.Type != TokEOF {
		t.Fatalf("last token is not EOF")
	}
}
