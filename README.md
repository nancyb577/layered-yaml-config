# layered-yaml-config

Most Go projects that need to read a YAML config file reach for
`gopkg.in/yaml.v3`. That's a fine library, but it's a dependency for
something a lot of config files don't actually need: the full YAML spec,
with anchors, aliases, flow style, multi-document streams, and all the
edge cases that come with them.

This is a much smaller thing: a parser for a restricted subset of YAML
(nested maps and scalars) written against nothing but the Go standard
library, plus a CLI for the two things I actually do with config files —
merge a base file with environment-specific overrides, and pull one value
out of a file without writing a one-off script.

## What it parses

```yaml
service:
  name: checkout
  port: 8080
  retries: 3
  timeout: 2.5
  debug: false
database:
  host: db.internal
  password: "quoted because it has a colon: in it"
cors_origins:
  - https://example.com
  - https://staging.example.com
```

Keys are plain strings, indentation defines nesting (spaces only, no
tabs), and scalar values are typed automatically as bool, int, float,
null, or string. Lines starting with `#` and blank lines are ignored,
and a `#` after a value starts a trailing comment (`port: 8080 # prod`)
unless it's inside a quoted string.

Double-quoted strings support the usual backslash escapes (`\n`, `\t`,
`\"`, `\\`, `\xNN`, `\uNNNN`, and so on); single-quoted strings don't —
use a doubled quote (`''`) to put a literal `'` inside one.

Lists are block-style only (`- value`, one per line, indented under the
key) and hold scalars, not nested maps or lists. Flow-style `{}`/`[]`,
lists of maps, anchors/aliases, and multi-line strings are not
supported. If a file uses them, `Parse` returns an error rather than
guessing.

## Library usage

```go
package main

import (
	"fmt"
	"log"

	yamlconf "github.com/nancyb577/layered-yaml-config"
)

func main() {
	base, err := yamlconf.ParseFile("config/base.yaml")
	if err != nil {
		log.Fatal(err)
	}
	prod, err := yamlconf.ParseFile("config/production.yaml")
	if err != nil {
		log.Fatal(err)
	}

	cfg := yamlconf.Merge(base, prod)
	fmt.Println(cfg["service"].(map[string]interface{})["port"])
}
```

`Merge(base, override)` layers `override` on top of `base`: nested maps
are merged key by key, and any other value in `override` replaces the
base value outright.

## Env var interpolation

A string scalar can reference an environment variable with `${VAR}`:

```yaml
database:
  host: db.internal
  password: ${DB_PASSWORD}
```

`ExpandEnv(cfg)` walks a parsed map and substitutes each `${VAR}`
reference with `os.LookupEnv(VAR)`, returning a new map. A `${VAR}` whose
variable isn't set is an error rather than a silent empty string, named
along with the dotted path of the value it appeared in (e.g.
`database.password: environment variable "DB_PASSWORD" is not set`).
`Expand(cfg, lookup)` takes a `func(string) (string, bool)` instead, for
looking values up somewhere other than the environment.

The CLI's `get` and `merge` subcommands call `ExpandEnv` on their result
before printing it, so `${VAR}` works there without extra flags.

## CLI usage

```
$ go run ./cmd/yamlconf merge config/base.yaml config/production.yaml
database:
  host: db.internal
  password: "quoted because it has a colon: in it"
service:
  debug: false
  name: checkout
  port: 8080
  retries: 3
  timeout: 2.5

$ go run ./cmd/yamlconf get config/base.yaml service.port
8080
```

## Status

This is early. The parser only handles the subset described above:
maps, block-style scalar lists, typed scalars, and quoted-string
escapes. There's no `validate` subcommand yet for checking a merged
config against a schema.
