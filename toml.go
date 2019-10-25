package toml

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

const tomlTag = "toml"

var (
	ErrSyntax    = errors.New("invalid syntax")
	ErrDuplicate = errors.New("duplicate")
	ErrUnknown   = errors.New("unknown")
)

func DecodeFile(file string, v interface{}) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()
	return Decode(r, v)
}

func Decode(r io.Reader, v interface{}) error {
	n, err := Parse(r)
	if err != nil {
		return err
	}
	root, ok := n.(*table)
	if !ok {
		return fmt.Errorf("root node is not a table!") // should never happen
	}
	e := reflect.ValueOf(v)
	if e.Kind() != reflect.Ptr || e.IsNil() {
		return fmt.Errorf("invalid given type %s", e.Type())
	}
	if e.Kind() == reflect.Interface && e.NumMethod() == 0 {
		m := make(map[string]interface{})
		me := reflect.ValueOf(m).Elem()
		if err = decodeMap(root, me); err == nil {
			e.Set(me)
		}
	} else {
		err = decodeTable(root, e.Elem())
	}
	return err
}

func decodeTableArray(t *table, v reflect.Value) error {
	if k := v.Kind(); k != reflect.Slice {
		return fmt.Errorf("expected slice, got %s", k)
	}
	var err error
	for i := range t.nodes {
		n, ok := t.nodes[i].(*table)
		if !ok {
			err = fmt.Errorf("unexpected table type: %T", t.nodes[i])
			break
		}
		f := reflect.New(v.Type().Elem()).Elem()
		if err = decodeTable(n, f); err != nil {
			break
		}
		v.Set(reflect.Append(v, f))
	}
	return err
}

func decodeTable(t *table, v reflect.Value) error {
	var err error
	switch k := v.Kind(); k {
	case reflect.Ptr:
		if v.IsNil() {
			f := reflect.New(v.Type().Elem())
			if err = decodeTable(t, reflect.Indirect(f)); err == nil {
				v.Set(f)
			}
		} else {
			err = decodeTable(t, v.Elem())
		}
	case reflect.Struct:
		err = decodeStruct(t, v)
	case reflect.Map:
		k := v.Type().Key()
		if k.Kind() != reflect.String {
			err = fmt.Errorf("map key should be of type string")
		} else {
			err = decodeMap(t, v)
		}
	default:
		err = fmt.Errorf("can not decode table into %s", k)
	}
	return err
}

func decodeMap(t *table, v reflect.Value) error {
	// for i := range t.nodes {
	// 	switch n := t.nodes[i].(type) {
	// 	case *table:
	// 	case option:
	// 	}
	// }
	return nil
}

func decodeStruct(t *table, v reflect.Value) error {
	sort.Slice(t.nodes, func(i, j int) bool {
		pi, pj := t.nodes[i].Pos(), t.nodes[j].Pos()
		return pi.Line <= pj.Line
	})

	obj, err := newObject(v)
	if err != nil {
		return err
	}
	for i := range t.nodes {
		switch n := t.nodes[i].(type) {
		case option:
			v, err = obj.Get(n.key.Literal)
			if err != nil {
				break
			}
			err = decodeOption(n, v)
		case *table:
			v, err = obj.Get(n.key.Literal)
			if err != nil {
				break
			}
			if n.kind == arrayTable {
				err = decodeTableArray(n, v)
			} else {
				err = decodeTable(n, v)
			}
		}
		if err != nil {
			break
		}
	}
	return err
}

func decodeArrayOption(a *array, v reflect.Value) error {
	if k := v.Kind(); k != reflect.Slice {
		return fmt.Errorf("expected slice, got %s", k)
	}
	var err error
	for i := range a.nodes {
		f := reflect.New(v.Type().Elem()).Elem()
		switch x := a.nodes[i].(type) {
		case literal:
			err = decodeLiteral(x, f)
		case *array:
			err = decodeArrayOption(x, f)
		case *table:
			err = decodeTable(x, f)
		}
		if err != nil {
			break
		}
		v.Set(reflect.Append(v, f))
	}
	return err
}

func decodeOption(o option, v reflect.Value) error {
	var err error
	switch x := o.value.(type) {
	case literal:
		err = decodeLiteral(x, v)
	case *array:
		err = decodeArrayOption(x, v)
	case *table:
		err = decodeTable(x, v)
	}
	return err
}

func decodeLiteral(lit literal, v reflect.Value) error {
	switch typ, kind := lit.token.Type, v.Kind(); {
	case typ == String && kind == reflect.String:
		v.SetString(lit.token.Literal)
	case typ == Integer && isInt(kind):
		x, err := strconv.ParseInt(lit.token.Literal, 0, 64)
		if err != nil {
			return err
		}
		v.SetInt(x)
	case typ == Integer && isUint(kind):
		x, err := strconv.ParseUint(lit.token.Literal, 0, 64)
		if err != nil {
			return err
		}
		v.SetUint(x)
	case typ == Float && isFloat(kind):
		x, err := strconv.ParseFloat(lit.token.Literal, 64)
		if err != nil {
			return err
		}
		v.SetFloat(x)
	case typ == Bool && kind == reflect.Bool:
		x, err := strconv.ParseBool(lit.token.Literal)
		if err != nil {
			return err
		}
		v.SetBool(x)
	case typ == DateTime && isTime(v):
		pattern := makeTimePatterns([]string{dtFormat1, dtFormat2}, true)
		t, err := parseTime(lit.token.Literal, pattern)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(t))
	case typ == Date && isTime(v):
		t, err := parseTime(lit.token.Literal, []string{dateFormat})
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(t))
	case typ == Time && isTime(v):
		pattern := makeTimePatterns([]string{timeFormat}, true)
		t, err := parseTime(lit.token.Literal, pattern)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(t))
	default:
		return fmt.Errorf("can not decode %s in %s (%s)", lit.token, kind, tokenString(typ))
	}
	return nil
}

var (
	tzFormat   = "Z07:00"
	dateFormat = "2006-01-02"
	timeFormat = "15:04:05"
	dtFormat1  = dateFormat + "T" + timeFormat
	dtFormat2  = dateFormat + " " + timeFormat
	millisPrec = ".000"
	microsPrec = ".000000"
)

func makeTimePatterns(pattern []string, zone bool) []string {
	ps := make([]string, 0, len(pattern)*4)
	millis := []string{millisPrec, microsPrec}
	for _, p := range pattern {
		ps = append(ps, p)
		for _, m := range millis {
			ps = append(ps, p+m)
			if zone {
				ps = append(ps, p+m+tzFormat)
			}
		}
	}
	return ps
}

func parseTime(str string, pattern []string) (time.Time, error) {
	for _, pat := range pattern {
		t, err := time.Parse(pat, str)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%s: no format match given datetime", str)
}

func isTime(v reflect.Value) bool {
	var z time.Time
	return v.Type().AssignableTo(reflect.TypeOf(z))
}

func isFloat(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

func isInt(k reflect.Kind) bool {
	return k == reflect.Int64 || k == reflect.Int32 || k == reflect.Int16 || k == reflect.Int8 || k == reflect.Int
}

func isUint(k reflect.Kind) bool {
	return k == reflect.Uint64 || k == reflect.Uint32 || k == reflect.Uint16 || k == reflect.Uint8 || k == reflect.Uint
}

type object struct {
	typ    reflect.Type
	fields []reflect.Value
	index  map[string]int
}

func newObject(v reflect.Value) (*object, error) {
	if k := v.Kind(); k != reflect.Struct {
		return nil, fmt.Errorf("newObject: expected struct, got %s", k)
	}
	obj := object{
		typ:    v.Type(),
		fields: make([]reflect.Value, 0, v.NumField()),
		index:  make(map[string]int),
	}
	typ := v.Type()
	for i, j := 0, 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}
		f := typ.Field(i)

		switch tag := f.Tag.Get(tomlTag); tag {
		case "":
		case "-":
			continue
		default:
			obj.index[strings.ToLower(tag)] = j
		}
		obj.index[strings.ToLower(f.Name)] = j
		obj.fields = append(obj.fields, field)

		j++
	}
	return &obj, nil
}

func (o object) Get(str string) (v reflect.Value, err error) {
	i, ok := o.index[str]
	if ok {
		v = o.fields[i]
	} else {
		err = fmt.Errorf("%s: table/option %w", str, ErrUnknown)
	}
	return
}
