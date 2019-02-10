package toml

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

type addr net.TCPAddr

func (a *addr) Set(v string) error {
	t, err := net.ResolveTCPAddr("tcp", v)
	if err != nil {
		return err
	}
	*a = addr(*t)
	return nil
}

type user struct {
	Name   string `toml:"name"`
	Passwd string `toml:"passwd"`
}

type pool struct {
	Proxy string `toml:"proxy"`
	User  user   `toml:"auth"`
	Hosts []conn `toml:"databases"`
}

type conn struct {
	Server  string   `toml:"server"`
	Ports   []uint16 `toml:"ports"`
	Limit   uint16   `toml:"limit"`
	Enabled bool     `toml:"enabled"`
	User    user     `toml:"auth"`
}

// func TestDecodeSetterValue(t *testing.T) {
// 	s := `
// addr = "127.0.0.1:80"
// 	`
// 	c := struct {
// 		Addr *addr `toml:"addr"`
// 	}{}
// 	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
// 		t.Fatal(err)
// 	}
// }

func TestDecodeSkipField(t *testing.T) {
	s := `
name = "TestDecodeSkipField"
	`
	c := struct {
		Name    string `toml:"name"`
		Package string `toml:"-"`
	}{Package: "toml"}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Fatal(err)
	}
	if c.Package != "toml" {
		t.Fatal("Package field has been modified")
	}
}

func TestDecodeQuottedKey(t *testing.T) {
	s := `
"double" = "quotation mark"
'single' = "single quote"
	`
	c := struct {
		Double string `toml:"double"`
		Single string `toml:"single"`
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Logf("%+v", c)
		t.Fatal(err)
	}
}

func TestDecodeDatetime(t *testing.T) {
	s := `
odt1 = 1979-05-27T07:32:00Z
odt2 = 1979-05-27T00:32:00-07:00
odt3 = 1979-05-27T00:32:00.999999-07:00
  `
	c := struct {
		Odt1 time.Time `toml:"odt1"`
		Odt2 time.Time `toml:"odt2"`
		Odt3 time.Time `toml:"odt3"`
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Fatal(err)
	}
}

func TestDecodePointer(t *testing.T) {
	s := `
address = "localhost:1024"

[[channels]]
server = "10.0.1.1:22"
enabled = false
auth = {name = "hello", passwd = "helloworld"}

[[channels]]
server = "10.0.1.2:22"
enabled = false
auth = {name = "hello", passwd = "helloworld"}
	`
	c := struct {
		Addr  string  `toml:"address"`
		Hosts []*conn `toml:"channels"`
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Fatal(err)
	}
}

func TestDecoderCompositeValues(t *testing.T) {
	s := `
group = "224.0.0.1"
ports = [[31001, 31002], [32001, 32002]]
auth = [
  {name = "midbel", passwd = "midbelrules101"},
  {name = "toml", passwd = "tomlrules101"},
]
  `
	c1 := struct {
		Group string
		Ports [][]int64
		Auth  []user
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c1); err != nil {
		t.Fatal(err)
	}
	// c2 := struct {
	// 	Group string
	// 	Ports [][]int64
	// 	Auth  []map[string]interface{}
	// }{}
	// if err := NewDecoder(strings.NewReader(s)).Decode(&c2); err != nil {
	// 	t.Fatal(err)
	// }
}

func TestDecoderKeyTypes(t *testing.T) {
	s := `
group = "224.0.0.1"
31001 = 31002
"enabled" = true
  `
	c := struct {
		Host   string `toml:"group"`
		Port   int64  `toml:"31001"`
		Active bool   `toml:"enabled"`
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeNestedTables(t *testing.T) {
	s := `
title = "TOML parser test"

[pool]
proxy = "192.168.1.124"
auth = {name = "midbel", passwd = "midbeltoml101"}

[[pool.databases]]
server = "192.168.1.1"
ports = [8001, 8002, 8003]
limit = 10

[[pool.databases]]
server = "192.168.1.1"
ports = [8001, 8002, 8003]
limit = 10
auth = {name = "midbel", passwd = "tomlrules101"}
  `
	c := struct {
		Title string `toml:"title"`
		Pool  pool   `toml:"pool"`
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeTableArray(t *testing.T) {
	s := `
title = "TOML parser test"

[[database]]
server = "192.168.1.1"
ports = [8001, 8002, 8003]
limit = 10

[[database]]
server = "192.168.1.1"
ports = [8001, 8002, 8003]
limit = 10
auth = {name = "midbel", passwd = "tomlrules101"}
  `
	c := struct {
		Title    string `toml:"title"`
		Settings []conn `toml:"database"`
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeSimple(t *testing.T) {
	s := `
title = "TOML parser test"

[database]
server = "192.168.1.1"
ports = [ 8001, 8001, 8002 ]
limit = 5000
enabled = true
  `
	c := struct {
		Title    string `toml:"title"`
		Settings conn   `toml:"database"`
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeMultilineString(t *testing.T) {
	d := `this is an extended description for the toml linter

* parse toml file
* show syntax error and provide ticks to fix them
* show syntax warning and provide tricks to solve them
* really configurable

enjoy the toml linter and the toml format
	`
	s := `
name = "tomllint"
version = "0.0"
summary = "a linter for toml files"
description = """%s"""
	`
	c := struct {
		Name        string `toml:"name"`
		Release     string `toml:"version"`
		Summary     string `toml:"summary"`
		Description string `toml:"description"`
	}{}
	s = fmt.Sprintf(s, d)
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Fatal(err)
	}
	if c.Description != d {
		t.Fatal("descriptions does not match")
	}
}

func TestDecodeIntoMap(t *testing.T) {
	s := `
title = "hello world"
date  = 2019-02-10T15:40:00Z
number = 10
	`
	var c map[string]interface{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
		t.Fatal(err)
	}
	if v, ok := c["title"]; ok {
		_, ok := v.(string)
		if !ok {
			t.Fatal("title string not set")
		}
	}
	if v, ok := c["date"]; ok {
		_, ok := v.(time.Time)
		if !ok {
			t.Fatal("date time not set")
		}
	}
	if v, ok := c["number"]; ok {
		_, ok := v.(int)
		if !ok {
			t.Fatal("number int not set")
		}
	}
}
