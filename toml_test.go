package toml

import (
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
	if a == nil {
		a = new(addr)
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

func TestDecoderSetterValue(t *testing.T) {
	s := `
addr = "127.0.0.1:80"
	`
	c := struct {
		Addr *addr `toml:"addr"`
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c); err != nil {
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
	c2 := struct {
		Group string
		Ports [][]int64
		Auth  []map[string]interface{}
	}{}
	if err := NewDecoder(strings.NewReader(s)).Decode(&c2); err != nil {
		t.Fatal(err)
	}
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
