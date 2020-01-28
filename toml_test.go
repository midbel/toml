package toml

import (
	"os"
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
	r, err := os.Open("testdata/package.toml")
	if err != nil {
		return err
	}
	defer r.Close()
	return Decode(r, p)
}
