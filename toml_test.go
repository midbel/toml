package toml

import (
	"strings"
	"testing"
	"time"
)

type Dependency struct {
	Repository string
	Version    string
}

type Changelog struct {
	Author string
	Text   string    `toml:"desc"`
	When   time.Time `toml:"date"`
}

type Dev struct {
	Name     string
	Email    string
	Projects []Project `toml:"project"`
}

type Project struct {
	Repo    string `toml:"repository"`
	Version string
	Active  bool
}

type Package struct {
	Name     string `toml:"package"`
	Version  string
	Repo     string `toml:"repository"`
	Provides []string
	Revision int

	Dev

	Deps []Dependency `toml:"dependency"`
	Logs []Changelog  `toml:"changelog"`
}

func TestDecode(t *testing.T) {
	t.Run("values", testDecodeValues)
	t.Run("pointers", testDecodePointers)
	t.Run("interface", testDecodeInterface)
	t.Run("map", testDecodeMap)
	t.Run("mix", testDecodeMix)
}

func testDecodeMix(t *testing.T) {
	const sample = `
nested = [[1, 2, 3], [3.14, 15.6, 0.18]]
inline = [
	{label="french", code="fr", enabled=false, translations=["français", "frans"]},
	{label="english", code="en", enabled=true, translations=["anglais", "engels"]},
	{label="dutch", code="nl", enabled=false, translations=["néerlandais", "nederlands"]},
]
table = {"url"= "/", title="welcome"}
dt1  = 2011-06-11T15:00:00+02:00
dt2  = 2011-06-11 15:00:00.000Z
dt3  = 2011-06-11 15:00:00.000123Z
	`
	var m interface{}
	if err := Decode(strings.NewReader(sample), &m); err != nil {
		t.Fatal(err)
	}
}

func testDecodeMap(t *testing.T) {
	m := make(map[string]interface{})
	if err := decodeFile(&m); err != nil {
		t.Fatal(err)
	}
}

func testDecodeInterface(t *testing.T) {
	var m interface{}
	if err := decodeFile(&m); err != nil {
		t.Fatal(err)
	}
}

func testDecodeValues(t *testing.T) {
	p := struct {
		Name     string `toml:"package"`
		Version  string
		Repo     string `toml:"repository"`
		Provides []string
		Revision int

		Dev

		Deps []Dependency `toml:"dependency"`
		Logs []Changelog  `toml:"changelog"`
	}{}
	if err := decodeFile(&p); err != nil {
		t.Fatal(err)
	}
}

func testDecodePointers(t *testing.T) {
	p := struct {
		Name     string `toml:"package"`
		Version  string
		Repo     string `toml:"repository"`
		Provides []string
		Revision int

		*Dev

		Deps []*Dependency `toml:"dependency"`
		Logs []*Changelog  `toml:"changelog"`
	}{}
	if err := decodeFile(&p); err != nil {
		t.Fatal(err)
	}
}

func decodeFile(p interface{}) error {
	return DecodeFile("testdata/package.toml", p)
}
