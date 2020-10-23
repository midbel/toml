# TOML decoder for Go

This package provides a TOML parser/decoder for go. It has been written to offer
an interface almost similar to the one of the xml and json packages of the standard
library.

On top of that, this package/repository provides commands to manipulate TOML documents.

installation

```bash
$ go get github.com/midbel/toml
```

This package has no external dependencies (except the stdlib).

Consider the following TOML document:

```toml
addr    = "0.0.0.0:12345"
version = 3

[[client]]
user   = "user0"
passwd = "temp123;"
roles  = ["user", "admin"]

  [client.access]
  host    = "10.0.1.1:10015"
  request = 10

[[client]]
user   = "user1"
passwd = "temp321;"
roles  = ["guest"]

  [client.access]
  host    = "10.0.1.2:100015"
  request = 50

[[group]]
addr   = "224.0.0.1:31001"
digest = false

[[group]]
addr   = "239.192.0.1:31001"
digest = true
```

it can then be decoded with:

```go
  type Client struct {
    User string
    Passwd string
    Roles []string
    Access struct {
      Host string
      Req  int `toml:"request"`
    }
  }

  func loadConfig(file string) error {
    r, err := os.Open(file)
    if err != nil {
      return err
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
    }
    defer r.Close()
    cfg := struct{
      Address string `toml:"addr"`
      Version int
      Clients []Client `toml:"client"`
      Groups  []struct{
        Address string `toml:"addr"`
        Request int
      } `toml:"group"`
    }{}
    if err := toml.Decode(r, &cfg); err != nil {
      return err
    }
    // use cfg
  }

  // or
  func loadConfig (file string) (map[string]interface{}, error) {
    // without opening the file
    var doc map[string]interface{}
    if err := toml.DecodeFile(file, &doc); err != nil {
      return nil, err
    }
    return doc, nil
  }
```

### Commands

##### tomlfmt

the command help you to rewrite a toml document with you preferences given in the options.

Some of its options are:

* removing comments from document
* align/nest tables and sub tables
* transform inline table into regular tables
* change base of all integers
* change formatting of all floats
* keep format of values from original document
* choose indentation type (tab by default, number of space)

to use it:

```bash
# build
$ cd <path>
$ go build -o bin/tomlfmt .

# execute
$ tomlfmt [options] <document.toml>
```

##### tomldump

the command shows you how the parser will understand your document if it is valid.

to use it:

```bash
# build
$ cd <path>
$ go build -o bin/tomldump .

# execute
$ tomldump <document.toml>
```
