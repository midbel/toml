package toml

import (
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
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
	root, ok := n.(*Table)
	if !ok {
		return fmt.Errorf("root node is not a table!") // should never happen
	}
	e := reflect.ValueOf(v)
	if e.Kind() != reflect.Ptr || e.IsNil() {
		return fmt.Errorf("invalid given type %s", e.Type())
	}
	if e.Kind() == reflect.Interface && e.NumMethod() == 0 {
		var (
			m  = make(map[string]interface{})
			me = reflect.ValueOf(m).Elem()
		)
		if err = decodeMap(root, me); err == nil {
			e.Set(me)
		}
	} else {
		err = decodeTable(root, e.Elem())
	}
	return err
}

func decodeTable(t *Table, e reflect.Value) error {
	var err error
	switch k := e.Kind(); k {
	case reflect.Interface:
		var (
			m  = make(map[string]interface{})
			me = reflect.ValueOf(m)
		)
		err = decodeMap(t, me)
		if err == nil {
			e.Set(me)
		}
	case reflect.Struct:
		err = decodeStruct(t, e)
	case reflect.Map:
		err = decodeMap(t, e)
	case reflect.Ptr:
		if e.IsNil() {
			f := reflect.New(e.Type().Elem())
			if err = decodeTable(t, reflect.Indirect(f)); err == nil {
				e.Set(f)
			}
		} else {
			err = decodeTable(t, e.Elem())
		}
	default:
		err = fmt.Errorf("table: unexpected type %s", k)
	}
	return err
}

func decodeArrayTable(t *Table, e reflect.Value) error {
	if k := e.Kind(); !(k == reflect.Array || k == reflect.Slice) {
		return fmt.Errorf("array: expected array/slice, got %s", k)
	}
	for _, n := range t.nodes {
		x, ok := n.(*Table)
		if !ok {
			return fmt.Errorf("array: unexpected node type %T", n)
		}
		f := reflect.New(e.Type().Elem()).Elem()
		if err := decodeTable(x, f); err != nil {
			return err
		}
		e.Set(reflect.Append(e, f))
	}
	return nil
}

func decodeArrayOption(a *Array, e reflect.Value) error {
	if isInterface(e.Kind()) {
		var (
			s = reflect.SliceOf(e.Type())
			f = reflect.MakeSlice(s, 0, len(a.nodes))
		)
		f = reflect.New(f.Type()).Elem()
		err := decodeArrayOption(a, f)
		if err == nil {
			e.Set(f)
		}
		return err
	}
	if k := e.Kind(); !(k == reflect.Array || k == reflect.Slice) {
		return fmt.Errorf("array: expected array/slice, got %s", k)
	}
	var err error
	for _, n := range a.nodes {
		f := reflect.New(e.Type().Elem()).Elem()
		switch n := n.(type) {
		case *Table:
			err = decodeTable(n, f)
		case *Array:
			err = decodeArrayOption(n, f)
		case *Literal:
			err = decodeLiteral(n, f)
		default:
			err = fmt.Errorf("array: unexpected node type %T", n)
		}
		if err != nil {
			break
		}
		e.Set(reflect.Append(e, f))
	}
	return err
}

func decodeOption(o *Option, e reflect.Value) error {
	var err error
	switch n := o.value.(type) {
	case *Array:
		err = decodeArrayOption(n, e)
	case *Table:
		err = decodeTable(n, e)
	case *Literal:
		err = decodeLiteral(n, e)
	default:
		err = fmt.Errorf("option: unexpected node type %T", n)
	}
	return err
}

func decodeLiteral(i *Literal, e reflect.Value) error {
	var err error
	switch str := i.token.Literal; i.token.Type {
	default:
		err = fmt.Errorf("literal: unexpected token type: %s", i.token)
	case TokString:
		err = decodeString(e, str)
	case TokBool:
		err = decodeBool(e, str)
	case TokInteger:
		err = decodeInt(e, str)
	case TokFloat:
		err = decodeFloat(e, str)
	case TokDatetime:
		patterns := makePatterns([]string{dtFormat1, dtFormat2})
		err = decodeTime(e, str, patterns)
	case TokDate:
		err = decodeTime(e, str, []string{dateFormat})
	case TokTime:
		// err = decodeTime(e, str)
	}
	return err
}

func decodeTime(e reflect.Value, str string, patterns []string) error {
	var (
		when time.Time
		err  error
	)
	if e.Type().AssignableTo(reflect.TypeOf(when)) || isInterface(e.Kind()) {
		for _, p := range patterns {
			when, err = time.Parse(p, str)
			if err == nil {
				e.Set(reflect.ValueOf(when))
				break
			}
		}
		if when.IsZero() && err == nil {
			err = fmt.Errorf("time(%s): no patterns matched", str)
		}
		return err
	}
	if !isString(e.Kind()) {
		err = fmt.Errorf("time(%s): unsupported type %s", str, e.Type())
	} else {
		e.SetString(str)
	}
	return err
}

func decodeFloat(e reflect.Value, str string) error {
	str = strings.ReplaceAll(str, "_", "")

	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}
	switch k := e.Kind(); {
	case isString(k):
		e.SetString(str)
	case isInt(k):
		if err = checkIntRange(k, int64(val)); err != nil {
			break
		}
		e.SetInt(int64(val))
	case isUint(k):
		if err = checkUintRange(k, uint64(val)); err != nil {
			break
		}
		if val >= 0 {
			e.SetUint(uint64(val))
		} else {
			err = fmt.Errorf("float(%s): negative number to unsigned", str)
		}
	case isFloat(k):
		if err = checkFloatRange(k, val); err != nil {
			break
		}
		e.SetFloat(val)
	case isInterface(k):
		e.Set(reflect.ValueOf(val))
	default:
		err = fmt.Errorf("float(%s): unsupported type %s", str, k)
	}
	return err
}

func decodeInt(e reflect.Value, str string) error {
	str = strings.ReplaceAll(str, "_", "")

	val, err := strconv.ParseInt(str, 0, 64)
	if err != nil {
		return err
	}
	switch k := e.Kind(); {
	case isString(k):
		e.SetString(str)
	case isInt(k):
		if err = checkIntRange(k, val); err != nil {
			break
		}
		e.SetInt(val)
	case isUint(k):
		if err = checkUintRange(k, uint64(val)); err != nil {
			break
		}
		e.SetUint(uint64(val))
	case isFloat(k):
		if err = checkFloatRange(k, float64(val)); err != nil {
			break
		}
		e.SetFloat(float64(val))
	case isInterface(k):
		e.Set(reflect.ValueOf(val))
	default:
		err = fmt.Errorf("int(%s): unsupported type %s", str, k)
	}
	return err
}

func decodeBool(e reflect.Value, str string) error {
	val, err := strconv.ParseBool(str)
	if err != nil {
		return err
	}
	switch k := e.Kind(); {
	case isString(k):
		e.SetString(str)
	case isBool(k):
		e.SetBool(val)
	case isInterface(k):
		e.Set(reflect.ValueOf(val))
	default:
		err = fmt.Errorf("bool(%s): unsupported type %s", str, k)
	}
	return err
}

func decodeString(e reflect.Value, str string) error {
	var err error
	switch k := e.Kind(); {
	case isString(k):
		e.SetString(str)
	case isInterface(k):
		e.Set(reflect.ValueOf(str))
	default:
		err = fmt.Errorf("string(%s): unsupported type %s", str, k)
	}
	return err
}

func decodeMap(t *Table, e reflect.Value) error {
	key := e.Type().Key()
	if k := key.Kind(); !isString(k) {
		return fmt.Errorf("map: key should be of type string")
	}
	if e.IsNil() {
		m := reflect.MakeMap(e.Type())
		e.Set(m)
	}
	var err error
	for _, n := range t.nodes {
		var (
			f reflect.Value
			k string
		)
		switch n := n.(type) {
		case *Table:
			k = n.key.Literal
			if n.kind == tableArray {
				var (
					vs = make([]interface{}, 0, len(n.nodes))
					m  = reflect.MakeSlice(reflect.TypeOf(vs), 0, len(n.nodes))
				)
				f = reflect.New(m.Type()).Elem()
				err = decodeArrayTable(n, f)
			} else {
				f = reflect.MakeMap(e.Type())
				err = decodeMap(n, f)
			}
		case *Option:
			f, k = reflect.New(e.Type().Elem()).Elem(), n.key.Literal
			err = decodeOption(n, f)
		default:
			err = fmt.Errorf("map: unexpected node type %T", n)
		}
		if err != nil {
			break
		}
		e.SetMapIndex(reflect.ValueOf(k), f)
	}
	return err
}

func decodeStruct(t *Table, e reflect.Value) error {
	var (
		err    error
		fields = getFields(e)
	)
	for _, n := range t.nodes {
		switch n := n.(type) {
		case *Option:
			f, ok := fields[n.key.Literal]
			if !ok {
				err = fmt.Errorf("%s: invalid option", n.key.Literal)
				break
			}
			err = decodeOption(n, f)
		case *Table:
			f, ok := fields[n.key.Literal]
			if !ok {
				err = fmt.Errorf("%s: invalid table", n.key.Literal)
				break
			}
			if n.kind == tableArray {
				err = decodeArrayTable(n, f)
			} else {
				err = decodeTable(n, f)
			}
		default:
			err = fmt.Errorf("table: unexpected node type %T", n)
		}
		if err != nil {
			break
		}
	}
	return err
}

func getFields(v reflect.Value) map[string]reflect.Value {
	fs := make(map[string]reflect.Value)
	if v.Kind() != reflect.Struct {
		return fs
	}
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		var (
			tf  = typ.Field(i)
			tag string
		)
		switch tag = tf.Tag.Get("toml"); tag {
		case "-":
			continue
		case "":
			tag = strings.ToLower(tf.Name)
		default:
		}
		fs[tag] = f
	}
	return fs
}

func isString(k reflect.Kind) bool {
	return k == reflect.String
}

func isBool(k reflect.Kind) bool {
	return k == reflect.Bool
}

func isInt(k reflect.Kind) bool {
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
		k == reflect.Int32 || k == reflect.Int64
}

func checkIntRange(k reflect.Kind, val int64) error {
	var (
		ok  bool
		err error
	)
	switch k {
	case reflect.Int8:
		ok = val >= math.MinInt8 && val <= math.MaxInt8
	case reflect.Int16:
		ok = val >= math.MinInt16 && val <= math.MaxInt16
	case reflect.Int32:
		ok = val >= math.MinInt32 && val <= math.MaxInt32
	case reflect.Int64, reflect.Int:
		ok = val >= math.MinInt64 && val <= math.MaxInt64
	}
	if !ok {
		err = fmt.Errorf("%s(%d): out of range", k, val)
	}
	return err
}

func isUint(k reflect.Kind) bool {
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 ||
		k == reflect.Uint32 || k == reflect.Uint64
}

func checkUintRange(k reflect.Kind, val uint64) error {
	var (
		ok  bool
		err error
	)
	switch k {
	case reflect.Uint8:
		ok = val <= math.MaxUint8
	case reflect.Uint16:
		ok = val <= math.MaxUint16
	case reflect.Uint32:
		ok = val <= math.MaxUint32
	case reflect.Uint64, reflect.Uint:
		ok = val <= math.MaxUint64
	}
	if !ok {
		err = fmt.Errorf("%s(%d): out of range", k, val)
	}
	return err
}

func isFloat(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

func checkFloatRange(k reflect.Kind, val float64) error {
	var (
		ok  bool
		err error
	)
	switch k {
	case reflect.Float32:
		ok = val >= -math.MaxFloat32 && val <= math.MaxFloat32
	case reflect.Float64:
		ok = val >= -math.MaxFloat64 && val <= math.MaxFloat64
	}
	if !ok {
		err = fmt.Errorf("%s(%f): out of range", k, val)
	}
	return err
}

func isInterface(k reflect.Kind) bool {
	return k == reflect.Interface
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

func makePatterns(patterns []string) []string {
	ps := make([]string, 0, len(patterns)*4)
	millis := []string{millisPrec, microsPrec}
	for _, p := range patterns {
		ps = append(ps, p)
		ps = append(ps, p+tzFormat)
		for _, m := range millis {
			ps = append(ps, p+m)
			ps = append(ps, p+m+tzFormat)
		}
	}
	return ps
}
