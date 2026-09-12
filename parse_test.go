package yamlconf

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, src string) map[string]interface{} {
	t.Helper()
	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return m
}

func TestParseEmpty(t *testing.T) {
	m := mustParse(t, "")
	if len(m) != 0 {
		t.Fatalf("want empty map, got %v", m)
	}
}

func TestParseScalarTypes(t *testing.T) {
	m := mustParse(t, `
name: checkout
port: 8080
timeout: 2.5
debug: false
enabled: true
nothing: null
tilde: ~
blank:
`)
	want := map[string]interface{}{
		"name":    "checkout",
		"port":    8080,
		"timeout": 2.5,
		"debug":   false,
		"enabled": true,
		"nothing": nil,
		"tilde":   nil,
		"blank":   nil,
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("key %q: got %#v, want %#v", k, m[k], v)
		}
	}
}

func TestParseNestedMaps(t *testing.T) {
	m := mustParse(t, `
service:
  name: checkout
  port: 8080
database:
  host: db.internal
`)
	service, ok := m["service"].(map[string]interface{})
	if !ok {
		t.Fatalf("service is not a map: %#v", m["service"])
	}
	if service["name"] != "checkout" || service["port"] != 8080 {
		t.Errorf("service = %#v", service)
	}
	database, ok := m["database"].(map[string]interface{})
	if !ok || database["host"] != "db.internal" {
		t.Errorf("database = %#v", m["database"])
	}
}

func TestParseEmptyNestedMap(t *testing.T) {
	m := mustParse(t, `
service:
other: 1
`)
	service, ok := m["service"].(map[string]interface{})
	if !ok || len(service) != 0 {
		t.Errorf("service = %#v, want empty map", m["service"])
	}
}

func TestParseList(t *testing.T) {
	m := mustParse(t, `
cors_origins:
  - https://example.com
  - https://staging.example.com
counts:
  - 1
  - 2
  - 3
`)
	origins, ok := m["cors_origins"].([]interface{})
	if !ok || len(origins) != 2 {
		t.Fatalf("cors_origins = %#v", m["cors_origins"])
	}
	if origins[0] != "https://example.com" || origins[1] != "https://staging.example.com" {
		t.Errorf("cors_origins = %#v", origins)
	}
	counts, ok := m["counts"].([]interface{})
	if !ok || len(counts) != 3 {
		t.Fatalf("counts = %#v", m["counts"])
	}
	if counts[0] != 1 || counts[1] != 2 || counts[2] != 3 {
		t.Errorf("counts = %#v", counts)
	}
}

func TestParseDoubleQuotedEscapes(t *testing.T) {
	m := mustParse(t, `
plain: "line one\nline two"
tab: "a\tb"
quote: "she said \"hi\""
hex: "\x41"
unicode: "\u00e9"
colon: "has: a colon # and hash"
`)
	if m["plain"] != "line one\nline two" {
		t.Errorf("plain = %q", m["plain"])
	}
	if m["tab"] != "a\tb" {
		t.Errorf("tab = %q", m["tab"])
	}
	if m["quote"] != `she said "hi"` {
		t.Errorf("quote = %q", m["quote"])
	}
	if m["hex"] != "A" {
		t.Errorf("hex = %q", m["hex"])
	}
	if m["unicode"] != "\u00e9" {
		t.Errorf("unicode = %q", m["unicode"])
	}
	if m["colon"] != "has: a colon # and hash" {
		t.Errorf("colon = %q", m["colon"])
	}
}

func TestParseSingleQuoted(t *testing.T) {
	m := mustParse(t, `
name: 'it''s a test'
plain: 'no \n escapes here'
`)
	if m["name"] != "it's a test" {
		t.Errorf("name = %q", m["name"])
	}
	if m["plain"] != `no \n escapes here` {
		t.Errorf("plain = %q", m["plain"])
	}
}

func TestParseComments(t *testing.T) {
	m := mustParse(t, `
# a full-line comment
port: 8080 # trailing comment
password: "quoted # not a comment"
url: https://example.com/path#fragment
`)
	if m["port"] != 8080 {
		t.Errorf("port = %#v", m["port"])
	}
	if m["password"] != "quoted # not a comment" {
		t.Errorf("password = %q", m["password"])
	}
	if m["url"] != "https://example.com/path#fragment" {
		t.Errorf("url = %q", m["url"])
	}
}

func TestParseBlankLinesIgnored(t *testing.T) {
	m := mustParse(t, "\na: 1\n\n\nb: 2\n")
	if m["a"] != 1 || m["b"] != 2 {
		t.Errorf("m = %#v", m)
	}
}

func TestParseRejectsTabs(t *testing.T) {
	_, err := Parse(strings.NewReader("a:\n\tb: 1\n"))
	if err == nil || !strings.Contains(err.Error(), "tabs") {
		t.Fatalf("got err %v, want a tabs error", err)
	}
}

func TestParseRejectsMissingColon(t *testing.T) {
	_, err := Parse(strings.NewReader("not a key value line\n"))
	if err == nil || !strings.Contains(err.Error(), "expected") {
		t.Fatalf("got err %v, want an expected-colon error", err)
	}
}

func TestParseRejectsEmptyKey(t *testing.T) {
	_, err := Parse(strings.NewReader(": value\n"))
	if err == nil || !strings.Contains(err.Error(), "empty key") {
		t.Fatalf("got err %v, want an empty-key error", err)
	}
}

func TestParseRejectsListItemNotUnderKey(t *testing.T) {
	_, err := Parse(strings.NewReader("- 1\n- 2\n"))
	if err == nil || !strings.Contains(err.Error(), "list item") {
		t.Fatalf("got err %v, want a list-item error", err)
	}
}

func TestParseRejectsUnexpectedIndent(t *testing.T) {
	_, err := Parse(strings.NewReader("a:\n  b: 1\n    c: 2\n"))
	if err == nil || !strings.Contains(err.Error(), "unexpected indentation") {
		t.Fatalf("got err %v, want an unexpected-indentation error", err)
	}
}
