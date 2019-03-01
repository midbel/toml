package scan

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"unicode"
	"unicode/utf8"
)

const (
	EOF rune = -(iota + 1)
	Ident
	String
	Char
	Int
	Uint
	Float
	Decimal
	Punct
	DateTime
	Date
	Time
	Invalid
)

const (
	Dot                = '.'
	Equal              = '='
	Comma              = ','
	LeftCurlyBracket   = '{'
	RightCurlyBracket  = '}'
	LeftSquareBracket  = '['
	RightSquareBracket = ']'
)

const (
	underscore = '_'
	hash       = '#'
	minus      = '-'
	plus       = '+'
	space      = ' '
	colon      = ':'
	scolon     = ';'
	backspace  = '\b'
	tab        = '\t'
	nl         = '\n'
	squote     = '\''
	dquote     = '"'
	bslash     = '\\'
	carriage   = '\r'
	formfeed   = '\f'
)

type Scanner struct {
	Last   rune
	offset int
	buffer []byte

	token bytes.Buffer
	Position
}

func NewScanner(r io.Reader) *Scanner {
	var s Scanner
	s.Reset(r)

	return &s
}

func (s *Scanner) Text() string {
	return s.token.String()
}

func (s *Scanner) peek() rune {
	offset := s.offset
	for {
		r, z := utf8.DecodeRune(s.buffer[offset:])
		if r == utf8.RuneError {
			return EOF
		}
		offset += z
		switch {
		case isWhitespace(r):
		case r == hash:
			for r != nl {
				r, z := utf8.DecodeRune(s.buffer[offset:])
				if r == utf8.RuneError {
					return EOF
				}
				offset += z
			}
		default:
			return r
		}
	}
}

func (s *Scanner) Peek() rune {
	switch r := s.peek(); {
	case isString(r):
		return String
	case isIdent(r):
		return Ident
	case isDigit(r):
		return Decimal
	default:
		return r
	}
}

func (s *Scanner) Scan() rune {
	r := s.scanRune()
	switch {
	case isWhitespace(r):
		r = s.skipWhitespace()
	case isComment(r):
		r = s.skipComment()
	}

	s.Offset = s.offset - 1
	s.token.Reset()
	switch {
	case isIdent(r):
		s.Last = s.scanIdent(r)
		switch s.token.String() {
		case "nan", "+nan", "-nan", "inf", "+inf", "-inf":
			s.Last = Float
		default:
		}
	case isString(r):
		var (
			multi, empty bool
			k rune
		)
		// check if empty string or multiline
		if n := s.peek(); n == r {
			s.scanRune()
			if n := s.peek(); n == r {
				s.scanRune()
				k = s.skipWhitespace()
				multi = true
			} else {
				empty, s.Last = true, String
			}
		} else {
			k = s.scanRune()
		}
		if !empty {
			if r == squote {
				s.Last = s.scanLiteralString(k, multi)
			} else if r == dquote {
				s.Last = s.scanBasicString(k, multi)
			} else {
				s.Last = Invalid
			}
		}
	case isDigit(r) || r == minus:
		s.Last = s.scanDecimal(r)
	case r == plus:
		s.Last = s.scanDecimal(s.scanRune())
		if s.Last != Float {
			s.Last = Uint
		}
	default:
		s.Last = r
	}
	return s.Last
}

func (s *Scanner) scanBasicString(r rune, multi bool) rune {
	s.token.WriteRune(r)

	for {
		r = s.scanRune()
		switch r {
		case EOF:
			return Invalid
		case bslash:
			r = s.scanRune()
			if r == nl && multi {
				continue
			}
			r = escapeRune(r)
		}
		if r == dquote {
			break
		}
		s.token.WriteRune(r)
	}
	if multi {
		if r := s.skipQuotes(dquote, false); r == Invalid {
			return r
		}
	}
	return String
}

func (s *Scanner) scanLiteralString(r rune, multi bool) rune {
	s.token.WriteRune(r)

	for r = s.scanRune(); ; r = s.scanRune() {
		if unicode.IsControl(r) && r != tab && r != nl {
			return Invalid
		}
		if r == EOF {
			return Invalid
		}
		s.token.WriteRune(r)
		if r == squote {
			if multi {
				if n := s.peek(); n != squote {
					continue
				}
			}
			break
		}
	}
	if multi {
		if r := s.skipQuotes(squote, false); r == Invalid {
			return r
		}
	}
	return String
}

func escapeRune(r rune) rune {
	switch r {
	default:
		return r
	case 'n':
		return nl
	case 'f':
		return formfeed
	case 'b':
		return backspace
	case 't':
		return tab
	case 'r':
		return carriage
	case '\\':
		return bslash
	case dquote:
		return dquote
	}
}

func (s *Scanner) skipQuotes(q rune, trim bool) rune {
	for i := 0; i < 2; i++ {
		if r := s.scanRune(); r == nl || r == EOF {
			s.token.WriteRune(q)
			return EOF
		} else if r != q {
			return Invalid
		}
	}
	if !trim {
		return String
	}
	for {
		r := s.scanRune()
		if !isWhitespace(r) {
			s.token.WriteRune(r)
			break
		}
	}
	return String
}

func (s *Scanner) Reset(r io.Reader) (err error) {
	s.buffer, err = ioutil.ReadAll(r)
	if err == nil && len(s.buffer) == 0 {
		err = io.EOF
	}
	s.offset = 0
	return err
}

func (s *Scanner) scanNumber(r rune, accept func(rune) bool) rune {
	if accept == nil {
		accept = func(r rune) bool {
			return isDigit(r) || r == underscore
		}
	}
	if r != underscore {
		s.token.WriteRune(r)
	}
	for {
		r = s.scanRune()
		if !accept(r) {
			if r != EOF {
				s.offset -= utf8.RuneLen(r)
			}
			break
		}
		if !(r == plus || r == underscore) {
			s.token.WriteRune(r)
		}
	}
	return Int
}

func (s *Scanner) scanDecimal(r rune) rune {
	if r != underscore {
		s.token.WriteRune(r)
	}
	switch n := s.peek(); n {
	case 'x':
		return s.scanNumber(s.scanRune(), isHexRune)
	case 'o':
		return s.scanNumber(s.scanRune(), isOctalRune)
	case 'b':
		return s.scanNumber(s.scanRune(), isBinRune)
	}
	for {
		switch r = s.scanRune(); {
		case r == colon:
			return s.scanTime(r)
		case r == minus:
			return s.scanDate(r)
		case r == underscore:
		case isDigit(r):
			s.token.WriteRune(r)
		case r == Dot:
			s.token.WriteRune(r)
			s.scanNumber(s.scanRune(), nil)
			if r = s.scanRune(); r == 'e' || r == 'E' {
				s.scanNumber(r, func(r rune) bool {
					return isDigit(r) || r == minus || r == plus
				})
			} else {
				s.offset -= utf8.RuneLen(r)
			}
			return Float
		case r == 'e' || r == 'E':
			s.scanNumber(r, func(r rune) bool {
				return isDigit(r) || r == minus || r == plus
			})
			return Float
		default:
			s.offset -= utf8.RuneLen(r)
			return Int
		}
	}
}

func (s *Scanner) scanDate(r rune) rune {
	s.token.WriteRune(r)
	for {
		switch r = s.scanRune(); {
		case r == ' ' || r == 'T':
			r = s.scanTime(r)
			switch r {
			case 'Z':
				s.token.WriteRune(r)
			case minus:
				s.scanTime(r)
			case Time:
			default:
				return EOF
			}
			return DateTime
		case isDigit(r) || r == minus:
			s.token.WriteRune(r)
		case r == nl || r == EOF:
			return Date
		default:
			return EOF
		}
	}
}

func (s *Scanner) scanTime(r rune) rune {
	s.token.WriteRune(r)
	for {
		switch r = s.scanRune(); {
		case r == nl || r == EOF:
			s.offset -= utf8.RuneLen(r)
			return Time
		case isDigit(r) || r == Dot || r == colon:
			s.token.WriteRune(r)
		default:
			return r
		}
	}
}

func (s *Scanner) scanIdent(r rune) rune {
	s.token.WriteRune(r)
	for {
		if r = s.scanRune(); r == EOF {
			break
		}
		if !isIdentRune(r) {
			s.offset -= utf8.RuneLen(r)
			break
		}
		s.token.WriteRune(r)
	}
	return Ident
}

func (s *Scanner) scanRune() rune {
	if s.offset >= len(s.buffer) {
		return EOF
	}
	r, z := utf8.DecodeRune(s.buffer[s.offset:])
	if r == utf8.RuneError {
		return EOF
	}
	s.offset += z
	return r
}

func (s *Scanner) skipWhitespace() rune {
	for {
		r := s.scanRune()
		if r == EOF {
			return EOF
		}
		if !isWhitespace(r) {
			if isComment(r) {
				return s.skipComment()
			}
			return r
		}
	}
}

func (s *Scanner) skipComment() rune {
	for {
		r := s.scanRune()
		if r == EOF {
			return EOF
		}
		if r == nl {
			return s.skipWhitespace()
		}
	}
}

var tokenTypes = map[rune]string{
	EOF:      "eof",
	Ident:    "ident",
	String:   "string",
	Int:      "int",
	Uint:     "uint",
	Float:    "float",
	Decimal:  "decimal",
	DateTime: "datetime",
	Date:     "date",
	Time:     "time",
	Invalid:  "invalid",
}

func TokenString(r rune) string {
	v, ok := tokenTypes[r]
	if ok {
		return v
	}
	return fmt.Sprintf("%v", r)
}

type Position struct {
	Line   int
	Column int
	Offset int
}

func (p Position) String() string {
	return fmt.Sprintf("<%d:%d>", p.Line, p.Column)
}

func isComment(r rune) bool {
	return r == hash
}

func isString(r rune) bool {
	return r == squote || r == dquote
}

func isIdent(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isIdentRune(r rune) bool {
	return isIdent(r) || r == '-' || r == underscore || isDigit(r)
}

func isWhitespace(r rune) bool {
	return r == space || r == tab || r == nl || r == carriage
}

func isHexRune(r rune) bool {
	return r == underscore || ('0' <= r && r <= '9') || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'Z')
}

func isOctalRune(r rune) bool {
	return r == underscore || ('0' <= r && r <= '7')
}

func isBinRune(r rune) bool {
	return r == underscore || r == '0' || r == '1'
}
