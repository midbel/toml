package toml

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/midbel/toml/internal/scan"
)

const (
	eof     = scan.EOF
	dot     = scan.Dot
	lsquare = scan.LeftSquareBracket
	rsquare = scan.RightSquareBracket
	lcurly  = scan.LeftCurlyBracket
	rcurly  = scan.RightCurlyBracket
	equal   = scan.Equal
	comma   = scan.Comma
)

type UnknownError struct {
	elem, name string
}

func (u UnknownError) Error() string {
	return fmt.Sprintf("toml: %s not recognized: %q!", u.elem, strings.Trim(u.name, "\""))
}

type SyntaxError struct {
	want, got rune
}

func (s SyntaxError) Error() string {
	if s.want <= 0 && s.got <= 0 {
		return fmt.Sprintf("toml: invalid syntax")
	}
	w, g := scan.TokenString(s.want), scan.TokenString(s.got)
	return fmt.Sprintf("toml: invalid syntax! want %s but got %s", w, g)
}

type Unmarshaler interface {
	UnmarshalTOML(*Decoder) error
}

type UnmarshalerOption interface {
	UnmarshalOption(*Decoder) error
}

type Decoder struct {
	scanner *scan.Scanner
}

func Unmarshal(bs []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(bs)).Decode(v)
}

func NewDecoder(r io.Reader) *Decoder {
	s := scan.NewScanner(r)
	return &Decoder{s}
}

func (d *Decoder) Decode(v interface{}) error {
	e := reflect.ValueOf(v)
	if k := e.Kind(); k != reflect.Ptr {
		return fmt.Errorf("expected pointer! got %s", k)
	}
	return d.decode(e.Elem())
}

func (d *Decoder) DecodeElement(v interface{}) error {
	e := reflect.ValueOf(v)
	if k := e.Kind(); k != reflect.Ptr {
		return fmt.Errorf("expected pointer! got %s", k)
	}
	switch e := e.Elem(); d.scanner.Last {
	case scan.Ident:
		return d.decodeBody(e)
	case equal:
		return d.decodeOption(e)
	default:
		return fmt.Errorf("can only be called to decode table or option")
	}
}

func (d *Decoder) decode(v reflect.Value) error {
	d.scanner.Scan()
	if err := d.decodeBody(v); err != nil {
		return err
	}
	vs := options(v)
	for t := d.scanner.Scan(); t != scan.EOF; t = d.scanner.Scan() {
		if err := d.decodeElement(vs); err != nil {
			return err
		}
	}
	return nil
}

func (d *Decoder) decodeElement(vs map[string]reflect.Value) error {
	var (
		v  reflect.Value
		ok bool
	)
	for t := d.scanner.Last; t != rsquare && t != scan.EOF; t = d.scanner.Scan() {
		switch t {
		case scan.Ident:
			v, ok = vs[d.scanner.Text()]
			if !ok {
				return tableNotFound(d.scanner.Text())
			}
		case lsquare:
			continue
		case scan.Dot:
			if k := v.Kind(); k == reflect.Slice {
				if v.Len() == 0 {
					x := reflect.New(v.Type().Elem()).Elem()
					v.Set(reflect.Append(v, x))
				}
				v = v.Index(v.Len() - 1)
			}
			d.scanner.Scan()
			return d.decodeElement(options(v))
		default:
			return unexpectedToken(t)
		}
	}
	for t := d.scanner.Last; t == rsquare; t = d.scanner.Scan() {
	}
	if k := v.Kind(); k == reflect.Slice {
		x := reflect.New(v.Type().Elem()).Elem()
		defer appendValue(v, x)
		v = x
	}
	if v.CanInterface() && v.Type().Implements(unmarshalerType) {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return v.Interface().(Unmarshaler).UnmarshalTOML(d)
	}
	if v.CanAddr() {
		if v := v.Addr(); v.CanInterface() && v.Type().Implements(unmarshalerType) {
			return v.Interface().(Unmarshaler).UnmarshalTOML(d)
		}
	}
	return d.decodeBody(v)
}

func (d *Decoder) decodeBody(v reflect.Value) error {
	vs := options(v)
	for t := d.scanner.Last; t != lsquare && t != eof; t = d.scanner.Scan() {
		if t != scan.String && t != scan.Ident && t != scan.Int {
			return invalidSyntax(0, 0)
		}
		f, ok := vs[strings.Trim(d.scanner.Text(), "\"")]
		if !ok {
			return optionNotFound(d.scanner.Text())
		}
		if t := d.scanner.Scan(); t != equal {
			return invalidSyntax(t, equal)
		}
		if err := d.decodeOption(f); err != nil {
			return err
		}
	}
	return nil
}

func (d *Decoder) decodeOption(v reflect.Value) error {
	if v.CanInterface() && v.Type().Implements(unmarshalerOptionType) {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return v.Interface().(UnmarshalerOption).UnmarshalOption(d)
	}
	if v.CanAddr() {
		if v := v.Addr(); v.CanInterface() && v.Type().Implements(unmarshalerOptionType) {
			return v.Interface().(UnmarshalerOption).UnmarshalOption(d)
		}
	}
	var err error
	switch t := d.scanner.Scan(); t {
	case lsquare:
		err = parseInlineArray(d.scanner, v)
	case lcurly:
		err = parseInlineTable(d.scanner, v)
	default:
		err = parseSimple(d.scanner, v)
	}
	return err
}

func invalidSyntax(w, g rune) error {
	return SyntaxError{w, g}
}

func unexpectedToken(g rune) error {
	return fmt.Errorf("toml: invalid syntax! unexpected token %s", scan.TokenString(g))
}

func tableNotFound(n string) error {
	return UnknownError{"table", n}
}

func optionNotFound(n string) error {
	return UnknownError{"option", n}
}

var (
	unmarshalerType       = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	unmarshalerOptionType = reflect.TypeOf((*UnmarshalerOption)(nil)).Elem()
)

func options(v reflect.Value) map[string]reflect.Value {
	if k := v.Kind(); k == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if k := v.Kind(); k != reflect.Struct {
		return nil
	}
	fs := make(map[string]reflect.Value)
	for i, t := 0, v.Type(); i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		j := t.Field(i)
		switch n := j.Tag.Get("toml"); {
		case n == "":
			fs[strings.ToLower(j.Name)] = f
		case n != "":
			fs[n] = f
		}
	}
	return fs
}

func appendValue(a, v reflect.Value) {
	a.Set(reflect.Append(a, v))
}

func parseSimple(s *scan.Scanner, f reflect.Value) error {
	v := strings.Trim(s.Text(), "\"")
	switch t, k := s.Last, f.Kind(); {
	case t == scan.Ident && k == reflect.Bool:
		n, _ := strconv.ParseBool(v)
		f.SetBool(n)
	case t == scan.String && k == reflect.String:
		f.SetString(v)
	case t == scan.Int && isInt(k):
		n, _ := strconv.ParseInt(v, 0, 64)
		f.SetInt(n)
	case t == scan.Int && isUint(k):
		n, _ := strconv.ParseUint(v, 0, 64)
		f.SetUint(n)
	case t == scan.Float && isFloat(k):
		n, _ := strconv.ParseFloat(v, 64)
		f.SetFloat(n)
	case (t == scan.Date || t == scan.DateTime) && isTime(f):
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
		f.Set(reflect.ValueOf(t))
	default:
		return fmt.Errorf("toml: unsupported type: %s (%s)", scan.TokenString(s.Last), k)
	}
	return nil
}

func parseInlineArray(s *scan.Scanner, f reflect.Value) error {
	for t := s.Scan(); t != rsquare && t != eof; t = s.Scan() {
		if t == comma {
			continue
		}
		var err error
		x := reflect.New(f.Type().Elem()).Elem()
		switch t {
		case lcurly:
			err = parseInlineTable(s, x)
		case lsquare:
			err = parseInlineArray(s, x)
		default:
			err = parseSimple(s, x)
		}
		f.Set(reflect.Append(f, x))
		if err != nil {
			return err
		}
	}
	return nil
}

func parseInlineTable(s *scan.Scanner, f reflect.Value) error {
	vs := options(f)
	for t := s.Scan(); t != rcurly && t != eof; t = s.Scan() {
		if t == comma {
			continue
		}
		var err error

		x, ok := vs[s.Text()]
		if !ok {
			return fmt.Errorf("unknown option %q", s.Text())
		}
		if t := s.Scan(); t != equal {
			return fmt.Errorf("body: invalid syntax! got %c want %c", t, equal)
		}
		switch t := s.Scan(); t {
		case lcurly:
			err = parseInlineTable(s, x)
		case lsquare:
			err = parseInlineArray(s, x)
		default:
			err = parseSimple(s, x)
		}
		if err != nil {
			return err
		}
	}
	return nil
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

func isTime(v reflect.Value) bool {
	var z time.Time
	return v.Type().AssignableTo(reflect.TypeOf(z))
}
