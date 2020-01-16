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
	return nil
}

func decodeMap(t *Table, e reflect.Value) error {
	return nil
}

func decodeArray(t *Table, e reflect.Value) error {
	return nil
}
