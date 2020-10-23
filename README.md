# TOML decoder for Go

installation

```bash
$ go get github.com/midbel/toml
```

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
```
