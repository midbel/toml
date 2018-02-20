package toml

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"text/scanner"
)

const (
	dot                = '.'
	comma              = ','
	minus              = '-'
	equal              = '='
	hash               = '#'
	leftSquareBracket  = '['
	rightSquareBracket = ']'
	leftCurlyBracket   = '{'
	rightCurlyBracket  = '}'
)

type Setter interface {
	Set(string) error
}

var setter = reflect.TypeOf((*Setter)(nil)).Elem()

type MalformedError struct {
	item string
	want rune
	got  rune
}

func (m MalformedError) Error() string {
	if m.want != 0 {
		return fmt.Sprintf("%s: expected %q, got %s", m.item, scanner.TokenString(m.want), scanner.TokenString(m.got))
	} else {
		return fmt.Sprintf("%s: unexpected token %s", m.item, scanner.TokenString(m.got))
	}
}

type DuplicateError struct {
	item  string
	label string
}

func (d DuplicateError) Error() string {
	return fmt.Sprintf("duplicate %s: %s", d.item, d.label)
}

type UndefinedError struct {
	item     string
	label    string
	position scanner.Position
}

func (u UndefinedError) Error() string {
	return fmt.Sprintf("%s. undefined %s: %q", u.position, u.item, u.label)
}

var booleans = map[string]bool{
	"true":  true,
	"false": false,
}

type Decoder struct {
	lex *lexer
}

func Unmarshal(bs []byte, v interface{}) error {
	r := bytes.NewReader(bs)
	return NewDecoder(r).Decode(v)
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{start(r)}
}

func (d *Decoder) Decode(v interface{}) error {
	return parse(d.lex, reflect.ValueOf(v).Elem())
}

type lexer struct {
	*scanner.Scanner
	token rune
}

func (l *lexer) Token() string {
	return scanner.TokenString(l.token)
}

func (l *lexer) Text() string {
	return l.TokenText()
}

func (l *lexer) Scan() rune {
	l.token = l.Scanner.Scan()
	if l.token == hash {
		p := l.Scanner.Position
		for {
			if l.Position.Line > p.Line {
				break
			}
			l.token = l.Scan()
		}
	}
	return l.token
}

func start(r io.Reader) *lexer {
	s := new(scanner.Scanner)
	s.Init(bufio.NewReader(r))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanFloats | scanner.ScanInts
	return &lexer{Scanner: s}
}

func parse(lex *lexer, v reflect.Value) error {
	if t := lex.Scan(); t == scanner.Ident || t == scanner.String {
		if err := parseOptions(lex, v); err != nil {
			return err
		}
	}
	for t := lex.Scan(); t != scanner.EOF; t = lex.Scan() {
		if err := parseTable(lex, v); err != nil {
			return err
		}
	}
	return nil
}

func parseTable(lex *lexer, v reflect.Value) error {
	var k reflect.Kind
	switch lex.token {
	default:
		return MalformedError{item: "section", got: lex.token}
	case scanner.Ident:
		k = reflect.Struct
	case leftSquareBracket:
		k = reflect.Slice
		lex.Scan()
	}
	var ok bool
	for t := lex.token; t != rightSquareBracket; t = lex.Scan() {
		if t == dot {
			continue
		}
		v, ok = listFields(v)[lex.Text()]
		if !ok {
			return UndefinedError{"section", lex.Text(), lex.Position}
		}
		switch k := v.Kind(); {
		case k == reflect.Slice && v.IsNil():
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		case k == reflect.Map && v.IsNil():
			v.Set(reflect.MakeMap(v.Type()))
		}
	}
	if t := v.Kind(); t != k {
		return fmt.Errorf("wrong type: expected %s, got %s", k, t)
	}
	switch t := lex.Scan(); t {
	default:
		return MalformedError{item: "section", got: lex.token}
	case scanner.Ident:
		return parseOptions(lex, v)
	case rightSquareBracket:
		lex.Scan()

		var f, z reflect.Value
		if e := v.Type().Elem(); e.Kind() == reflect.Ptr {
			z = reflect.New(e.Elem())
			f = z.Elem()
		} else {
			f = reflect.New(e).Elem()
			z = f
		}
		if err := parseOptions(lex, f); err != nil {
			return err
		}
		v.Set(reflect.Append(v, z))
	}
	return nil
}

func parseOptions(lex *lexer, v reflect.Value) error {
	fs := listFields(v)
	if lex.token == leftSquareBracket || lex.token == scanner.EOF {
		return nil
	}
	for {
		if err := parseOption(lex, fs); err != nil {
			return err
		}
		if t := lex.Scan(); t == leftSquareBracket || t == scanner.EOF {
			break
		}
	}
	return nil
}

func parseOption(lex *lexer, fs map[string]reflect.Value) error {
	f, ok := fs[strings.Trim(lex.Text(), "\"")]
	if !ok {
		return UndefinedError{"option", lex.Text(), lex.Position}
	}
	if t := lex.Scan(); t != equal {
		return MalformedError{"option", equal, t}
	}
	if t := lex.Peek(); t == '\n' {
		return MalformedError{"option", scanner.Ident, t}
	}
	var err error
	switch t := lex.Scan(); t {
	case leftSquareBracket:
		err = parseArray(lex, f)
	case leftCurlyBracket:
		err = parseMap(lex, f)
	default:
		err = parseSimple(lex, f, false)
	}
	return err
}

func parseMap(lex *lexer, v reflect.Value) error {
	if k := v.Kind(); k != reflect.Struct {
		return fmt.Errorf("table: struct expected, got %s", k)
	}
	fs := listFields(v)
	for t := lex.Scan(); t != rightCurlyBracket; t = lex.Scan() {
		if t == comma {
			continue
		}
		if err := parseOption(lex, fs); err != nil {
			return err
		}
	}
	return nil
}

func parseArray(lex *lexer, f reflect.Value) error {
	if k := f.Kind(); k != reflect.Slice {
		return fmt.Errorf("array: slice expected, got %s", k)
	}
	for t := lex.Scan(); t != rightSquareBracket; t = lex.Scan() {
		var err error
		v := reflect.New(f.Type().Elem()).Elem()
		switch t {
		case comma:
			continue
		case leftSquareBracket:
			err = parseArray(lex, v)
		case leftCurlyBracket:
			err = parseMap(lex, v)
		default:
			err = parseSimple(lex, v, false)
		}
		if err != nil {
			return err
		}
		f.Set(reflect.Append(f, v))
	}
	return nil
}

func parseSimple(lex *lexer, f reflect.Value, r bool) error {
	if f.Type().Implements(setter) {
		if f.IsNil() {
			f.Set(reflect.New(f.Type().Elem()))
		}
		s := f.Interface().(Setter)
		return s.Set(strings.Trim(lex.Text(), "\""))
	}
	switch t, k := lex.token, f.Kind(); {
	case t == scanner.String && k == reflect.String:
		f.SetString(strings.Trim(lex.Text(), "\""))
	case t == scanner.Int && isUint(k):
		v, _ := strconv.ParseUint(lex.Text(), 0, 64)
		f.SetUint(v)
	case t == scanner.Int && isInt(k):
		v, _ := strconv.ParseInt(lex.Text(), 0, 64)
		if r {
			v = -v
		}
		f.SetInt(v)
	case t == scanner.Float && isFloat(k):
		v, _ := strconv.ParseFloat(lex.Text(), 64)
		if r {
			v = -v
		}
		f.SetFloat(v)
	case t == scanner.Ident && k == reflect.Bool:
		f.SetBool(booleans[lex.Text()])
	case t == minus && (isInt(k) || isFloat(k)):
		lex.Scan()
		return parseSimple(lex, f, true)
	default:
		return fmt.Errorf("option: can not assign %s to %s", lex.Text(), k)
	}
	return nil
}

func listFields(v reflect.Value) map[string]reflect.Value {
	fs := make(map[string]reflect.Value)
	for i, t := 0, v.Type(); i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		switch f.Kind() {
		case reflect.Interface:
			f = f.Elem().Elem()
		case reflect.Ptr:
			//			f = f.Elem()
		}
		z := t.Field(i)
		switch n := z.Tag.Get("toml"); n {
		case "-":
			continue
		case "":
			fs[strings.ToLower(z.Name)] = f
		default:
			fs[n] = f
		}
	}
	return fs
}

func isInt(k reflect.Kind) bool {
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

func isUint(k reflect.Kind) bool {
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64
}

func isFloat(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}
