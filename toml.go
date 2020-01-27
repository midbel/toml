package toml

import (
	"fmt"
	"io"
	"os"
	"reflect"
)

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

func DecodeFile(file string, v interface{}) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	defer r.Close()
	return Decode(r, v)
}

func decodeTable(t *Table, e reflect.Value) error {
	var err error
	for _, n := range t.nodes {
		switch n := n.(type) {
		case *Option:
			err = decodeOption(n, e)
		case *Table:
			err = decodeTable(n, e)
		default:
			err = fmt.Errorf("table: unexpected node type %T", n)
		}
		if err != nil {
			break
		}
	}
	return nil
}

func decodeMap(t *Table, e reflect.Value) error {
	return nil
}

func decodeArray(a *Array, e reflect.Value) error {
	if k := e.Kind(); k != reflect.Array || k != reflect.Slice {
		return fmt.Errorf("array: expected array/slice, got %s", k)
	}
	return nil
}

func decodeOption(o *Option, e reflect.Value) error {
	var err error
	switch n := o.value.(type) {
	case *Array:
		err = decodeArray(n, e)
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
	return nil
}
