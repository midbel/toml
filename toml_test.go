package toml

import (
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
