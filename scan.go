package toml

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"unicode/utf8"
)

const (
	carriage   = '\r'
	newline    = '\n'
	pound      = '#'
	space      = ' '
	tab        = '\t'
	equal      = '='
	dot        = '.'
	comma      = ','
	lsquare    = '['
	rsquare    = ']'
	lcurly     = '{'
	rcurly     = '}'
	plus       = '+'
	minus      = '-'
	underscore = '_'
	colon      = ':'
	backslash  = '\\'
)

var escapes = map[rune]rune{
	'b':  '\b',
	't':  tab,
	'n':  newline,
	'r':  carriage,
	'f':  '\f',
	'"':  '"',
	'\\': backslash,
}

type scanMode int8

const (
	scanKey scanMode = iota
	scanValue
)

type Scanner struct {
	buffer []byte
	pos    int
	next   int

	mode  scanMode
	stack uint

	char rune

	KeepComment   bool
	KeepMultiline bool

	line   int
	column int
	rowlen int
}

func Scan(r io.Reader) (*Scanner, error) {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	s := Scanner{
		buffer:        bytes.ReplaceAll(buf, []byte("\r\n"), []byte("\n")),
		line:          1,
		column:        0,
		KeepComment:   false,
		KeepMultiline: false,
		mode:          scanKey,
	}
	s.readRune()
	s.skipWith(isNewline)

	return &s, nil
}

func (s *Scanner) Scan() Token {
	var t Token
	if s.char == EOF {
		t.Type = EOF
		return t
	}
	s.skipBlank()

	pos := Position{
		Line:   s.line,
		Column: s.column,
	}
	s.switchMode()
	switch {
	default:
		t.Type = Illegal
	case isComment(s.char):
		s.scanComment(&t)
		s.readRune()
		if !s.KeepComment {
			return s.Scan()
		} else {
			s.readRune()
		}
	case isLetter(s.char) || (s.mode == scanKey && isDigit(s.char)):
		s.scanIdent(&t)
	case (s.mode == scanValue && isDigit(s.char)) || isSign(s.char):
		if accept := acceptBase(s.peekRune()); s.char == '0' && accept != nil {
			s.scanNumberWith(&t, accept)
			break
		}
		s.scanNumber(&t, isSign(s.char))
	case isQuote(s.char):
		s.scanString(&t)
	case isPunct(s.char):
		t.Type = s.char
	case isNewline(s.char):
		t.Type = Newline
		if peek := s.peekRune(); !s.KeepMultiline && peek == newline {
			s.readRune()
			return s.Scan()
		}
	}
	s.readRune()

	t.Pos = pos
	return t
}

func (s *Scanner) switchMode() {
	if isNewline(s.char) && s.mode == scanValue && s.stack == 0 {
		s.mode = scanKey
		return
	}
	switch {
	case s.mode == scanValue && (s.char == lcurly || s.char == lsquare):
		s.stack++
	case s.mode == scanValue && (s.char == rcurly || s.char == rsquare):
		s.stack--
	case s.mode == scanKey && s.char == equal:
		s.mode = scanValue
	}
}

func (s *Scanner) readRuneN(n int) int {
	var i int
	for ; i < n && s.char != EOF; i++ {
		s.readRune()
	}
	return i
}

func (s *Scanner) readRune() {
	if s.next >= len(s.buffer) {
		s.char = EOF
		return
	}
	r, n := utf8.DecodeRune(s.buffer[s.next:])
	if r == utf8.RuneError {
		if n == 0 {
			s.char = EOF
		} else {
			s.char = Illegal
		}
		s.next = len(s.buffer)
	}
	s.char, s.pos, s.next = r, s.next, s.next+n
	if s.char == newline {
		s.line++
		s.rowlen, s.column = s.column, 0
	} else {
		s.column++
	}
}

func (s *Scanner) unreadRune() {
	if s.next <= 0 || s.char == 0 {
		return
	}

	if s.char == newline {
		s.line--
		s.column = s.rowlen
	} else {
		s.column--
	}

	s.next, s.pos = s.pos, s.pos-utf8.RuneLen(s.char)
	s.char, _ = utf8.DecodeRune(s.buffer[s.pos:])
}

func (s *Scanner) peekRune() rune {
	if s.next >= len(s.buffer) {
		return EOF
	}
	r, n := utf8.DecodeRune(s.buffer[s.next:])
	if r == utf8.RuneError {
		if n == 0 {
			r = EOF
		} else {
			r = Illegal
		}
	}
	return r
}

func (s *Scanner) scanNumber(t *Token, signed bool) {
	t.Type = Integer

	eol := func() bool {
		return isDelim(s.char) || isWhitespace(s.char) || s.char == EOF
	}
	var (
		pos    = s.pos
		zeros  int
	)
	if sign := s.char; signed {
		s.readRune()
		if sign == plus {
			pos = s.pos
		}
	}
	for s.char == '0' {
		zeros++
		s.readRune()
	}

	if eol() {
		s.unreadRune()
		if zeros > 1 {
			t.Type = Illegal
		}
		t.Literal = string(s.buffer[pos : s.pos+1])
		return
	}

	for prev := s.char; !(t.Type == Illegal || eol()); {
		switch {
		case isDigit(s.char):
			prev = s.char
		case s.char == underscore:
			if !(isDigit(prev) && isDigit(s.peekRune())) {
				t.Type = Illegal
			}
		case s.char == dot:
			s.scanFraction(t)
		case s.char == 'e' || s.char == 'E':
			s.scanExponent(t)
		case s.char == minus:
			s.scanDate(t)
			if signed {
				t.Type = Illegal
			}
		case s.char == colon:
			s.scanTime(t)
			if signed {
				t.Type = Illegal
			}
		default:
			t.Type = Illegal
		}
		s.readRune()
	}

	if (t.Type == Integer && zeros > 0) || (t.Type == Float && zeros > 1) {
		t.Type = Illegal
	}

	s.unreadRune()
	t.Literal = string(s.buffer[pos : s.pos+1])
}

func (s *Scanner) scanDate(t *Token) int {
	t.Type = Date

	offset := s.readRuneN(3)
	if s.char != minus {
		t.Type = Illegal
		return offset
	}

	offset += s.readRuneN(3)
	switch {
	case s.char == 'T' || s.char == space:
		offset += s.scanTime(t)
		switch s.char {
		case 'Z':
			offset++
		case plus, minus:
			offset += s.scanTimezone(t)
		default:
		}
		if t.Type != Illegal {
			t.Type = DateTime
		}
	case isNewline(s.char) || isPunct(s.char) || s.char == EOF:
		s.unreadRune()
		t.Type = Date
	default:
		t.Type = Illegal
	}
	return offset
}

func (s *Scanner) scanTime(t *Token) int {
	t.Type = Time

	var offset int
	if s.char != colon {
		if offset += s.readRuneN(3); s.char != colon {
			t.Type = Illegal
			return offset
		}
	}

	if offset += s.readRuneN(3); s.char != colon {
		t.Type = Illegal
		return offset
	}

	if offset += s.readRuneN(3); s.char == dot {
		offset += s.scanMillis(t)
	}
	if isNewline(s.char) || isPunct(s.char) || s.char == EOF {
		s.unreadRune()
	}
	return offset
}

func (s *Scanner) scanMillis(t *Token) int {
	var offset int

	s.readRune()
	offset++
	for isDigit(s.char) {
		s.readRune()
		offset++
	}
	if offset < 3 {
		t.Type = Illegal
	}
	return offset
}

func (s *Scanner) scanTimezone(t *Token) int {
	offset := s.readRuneN(3)
	if s.char != colon {
		t.Type = Illegal
		return offset
	}
	offset += s.readRuneN(2)
	return offset + 1
}

func (s *Scanner) scanExponent(t *Token) int {
	t.Type = Float

	s.readRune() // consume the 'e' or the 'E'
	var (
		offset = utf8.RuneLen('e')
		prev   = s.char
	)
	if isSign(s.char) {
		s.readRune()
		offset += utf8.RuneLen(s.char)
	}
	for {
		if !isDigit(s.char) && s.char != underscore {
			t.Type = Illegal
			return offset
		}
		if s.char == underscore {
			if !(isDigit(prev) || isDigit(s.peekRune())) {
				t.Type = Illegal
				return offset
			}
		}
		offset += utf8.RuneLen(s.char)
		prev = s.char
		s.readRune()
		if isWhitespace(s.char) || s.char == EOF || isPunct(s.char) {
			s.unreadRune()
			break
		}
	}
	return offset
}

func (s *Scanner) scanFraction(t *Token) int {
	t.Type = Float
	var (
		prev   = s.char
		offset = 1
	)
	s.readRune() // consume the dot
Loop:
	for {
		switch {
		case isDigit(s.char):
			prev = s.char
			offset += utf8.RuneLen(s.char)
		case s.char == underscore:
			if !(isDigit(s.peekRune()) || isDigit(prev)) {
				t.Type = Illegal
				break Loop
			}
			offset++
		case s.char == 'e' || s.char == 'E':
			offset += s.scanExponent(t)
		case isPunct(s.char) || isWhitespace(s.char) || s.char == EOF:
			s.unreadRune()
			break Loop
		default:
			t.Type = Illegal
		}
		s.readRune()
	}
	return offset
}

func (s *Scanner) scanNumberWith(t *Token, accept func(rune) bool) {
	pos := s.pos

	s.readRune()
	s.readRune()

	eol := func() bool {
		return isDelim(s.char) || isWhitespace(s.char) || s.char == EOF
	}
	for prev := s.char; !(t.Type == Illegal || eol()); {
		if s.char == underscore && !(accept(prev) && accept(s.peekRune())) {
			t.Type = Illegal
		}
		prev = s.char
		s.readRune()
	}
	s.unreadRune()

	t.Type = Integer
	t.Literal = string(s.buffer[pos : s.pos+1])
}

func (s *Scanner) scanComment(t *Token) {
	t.Type = Comment

	s.readRune()
	s.skipBlank()

	var (
		pos    = s.pos
		offset int
	)
	for !isNewline(s.char) {
		s.readRune()
		offset += utf8.RuneLen(s.char)
	}
	s.unreadRune()
	t.Literal = string(s.buffer[pos : pos+offset])
}

func (s *Scanner) scanString(t *Token) {
	t.Type = String

	var (
		quote = s.char
		multi bool
		buf   strings.Builder
	)
	s.readRune()
	if s.char == quote {
		s.readRune()
		if !isQuote(s.char) {
			s.unreadRune()
			return
		}
		if multi = s.char == quote; multi {
			s.readRune()
			if isNewline(s.char) {
				s.readRune()
			}
		}
	}
	for s.char != quote {
		if quote == '"' && s.char == backslash {
			s.readRune()
			if isNewline(s.char) {
				s.skipWith(isWhitespace)
				continue
			}
			if char, ok := escapes[s.char]; !ok {
				t.Type = Illegal
				return
			} else {
				s.char = char
			}
		}
		if !utf8.ValidRune(s.char) {
			t.Type = Illegal
			return
		}
		buf.WriteRune(s.char)
		s.readRune()
	}
	t.Literal = buf.String()
	if multi {
		s.readRune()
		s.readRune()
	}
	if s.char != quote {
		t.Type = Illegal
		return
	}
}

func (s *Scanner) scanIdent(t *Token) {
	t.Type = Ident
	var (
		pos    = s.pos
		offset int
	)
	for isIdent(s.char) {
		s.readRune()
		offset += utf8.RuneLen(s.char)
	}
	t.Literal = string(s.buffer[pos : pos+offset])
	switch t.Literal {
	case "true", "false":
		t.Type = Bool
	case "inf", "nan":
		t.Type = Float
	}
	if s.mode == scanKey {
		t.Type = Ident
	}
	s.unreadRune()
}

func (s *Scanner) skipWith(is func(rune) bool) int {
	var i int
	for is(s.char) {
		s.readRune()
		i += utf8.RuneLen(s.char)
	}
	return i
}

func (s *Scanner) skipBlank() {
	s.skipWith(isBlank)
}

func acceptBase(r rune) func(rune) bool {
	var accept func(rune) bool
	switch r {
	case 'x':
		return isHexa
	case 'o':
		return isOctal
	case 'b':
		return isBinary
	}
	return accept
}

func isHexa(r rune) bool {
	return isDigit(r) || (r >= 'A' && r <= 'F') || (r >= 'a' && r <= 'f') || r == underscore
}

func isOctal(r rune) bool {
	return (r >= '0' && r <= '7') || r == underscore
}

func isBinary(r rune) bool {
	return r == '0' || r == '1' || r == underscore
}

func isDelim(r rune) bool {
	return r == rsquare || r == rcurly || r == comma
}

func isPunct(r rune) bool {
	return r == equal || r == dot || r == lsquare || r == rsquare || r == rcurly || r == lcurly || r == comma
}

func isIdent(r rune) bool {
	return isAlpha(r) || r == minus || r == underscore
}

func isAlpha(r rune) bool {
	return isDigit(r) || isLetter(r)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isSign(r rune) bool {
	return r == minus || r == plus
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isQuote(r rune) bool {
	return r == '\'' || r == '"'
}

func isComment(r rune) bool {
	return r == pound
}

func isBlank(r rune) bool {
	return r == space || r == tab
}

func isNewline(r rune) bool {
	return r == newline
}

func isWhitespace(r rune) bool {
	return isBlank(r) || isNewline(r)
}
