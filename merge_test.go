package yamlconf

import (
	"reflect"
	"strings"
	"testing"
)

func TestMergeNestedMaps(t *testing.T) {
	base := map[string]interface{}{
		"service": map[string]interface{}{
			"name": "checkout",
			"port": 8080,
		},
		"database": map[string]interface{}{
			"host": "db.internal",
		},
	}
	override := map[string]interface{}{
		"service": map[string]interface{}{
			"port": 9090,
		},
	}
	got := Merge(base, override)

	service := got["service"].(map[string]interface{})
	if service["name"] != "checkout" {
		t.Errorf("service.name = %#v, want unchanged base value", service["name"])
	}
	if service["port"] != 9090 {
		t.Errorf("service.port = %#v, want override value", service["port"])
	}
	database := got["database"].(map[string]interface{})
	if database["host"] != "db.internal" {
		t.Errorf("database.host = %#v, want untouched base value", database["host"])
	}
}

func TestMergeScalarReplacesMap(t *testing.T) {
	base := map[string]interface{}{
		"service": map[string]interface{}{"port": 8080},
	}
	override := map[string]interface{}{
		"service": "disabled",
	}
	got := Merge(base, override)
	if got["service"] != "disabled" {
		t.Errorf("service = %#v, want override scalar to replace the map", got["service"])
	}
}

func TestMergeMapReplacesScalar(t *testing.T) {
	base := map[string]interface{}{
		"service": "disabled",
	}
	override := map[string]interface{}{
		"service": map[string]interface{}{"port": 8080},
	}
	got := Merge(base, override)
	service, ok := got["service"].(map[string]interface{})
	if !ok || service["port"] != 8080 {
		t.Errorf("service = %#v, want override map to replace the scalar", got["service"])
	}
}

func TestMergeDisjointKeys(t *testing.T) {
	base := map[string]interface{}{"a": 1}
	override := map[string]interface{}{"b": 2}
	got := Merge(base, override)
	if got["a"] != 1 || got["b"] != 2 {
		t.Errorf("got = %#v", got)
	}
}

func TestMergeDoesNotMutateInputs(t *testing.T) {
	base := map[string]interface{}{
		"service": map[string]interface{}{"port": 8080},
	}
	override := map[string]interface{}{
		"service": map[string]interface{}{"port": 9090},
	}
	Merge(base, override)
	if base["service"].(map[string]interface{})["port"] != 8080 {
		t.Errorf("base was mutated: %#v", base)
	}
	if override["service"].(map[string]interface{})["port"] != 9090 {
		t.Errorf("override was mutated: %#v", override)
	}
}

func TestDumpSortsKeysAndNests(t *testing.T) {
	m := map[string]interface{}{
		"zebra": 1,
		"apple": 2,
		"nested": map[string]interface{}{
			"z": 1,
			"a": 2,
		},
	}
	got := Dump(m)
	want := "apple: 2\nnested:\n  a: 2\n  z: 1\nzebra: 1\n"
	if got != want {
		t.Errorf("Dump() =\n%q\nwant\n%q", got, want)
	}
}

func TestDumpQuotesAmbiguousStrings(t *testing.T) {
	m := map[string]interface{}{
		"looks_bool": "true",
		"looks_null": "null",
		"has_colon":  "a: b",
		"has_space":  " padded ",
		"plain":      "checkout",
	}
	got := Dump(m)
	for key, want := range map[string]string{
		"looks_bool": `looks_bool: "true"` + "\n",
		"looks_null": `looks_null: "null"` + "\n",
		"has_colon":  `has_colon: "a: b"` + "\n",
		"has_space":  `has_space: " padded "` + "\n",
		"plain":      "plain: checkout\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("key %q: Dump() = %q, want it to contain %q", key, got, want)
		}
	}
}

func TestDumpList(t *testing.T) {
	// Both URLs contain a colon, so needsQuoting forces them to be quoted
	// in the output even though they parse fine unquoted.
	m := map[string]interface{}{
		"origins": []interface{}{"https://example.com", "https://staging.example.com"},
	}
	want := "origins:\n  - \"https://example.com\"\n  - \"https://staging.example.com\"\n"
	if got := Dump(m); got != want {
		t.Errorf("Dump() = %q, want %q", got, want)
	}
}

func TestParseDumpRoundTrip(t *testing.T) {
	m := map[string]interface{}{
		"name":    "checkout",
		"port":    8080,
		"timeout": 2.5,
		"debug":   false,
		"nothing": nil,
		"looks_bool": "true",
		"service": map[string]interface{}{
			"host": "db.internal",
		},
		"origins": []interface{}{"https://example.com", "https://staging.example.com"},
	}
	roundTripped, err := Parse(strings.NewReader(Dump(m)))
	if err != nil {
		t.Fatalf("Parse(Dump(m)): %v", err)
	}
	if !reflect.DeepEqual(m, roundTripped) {
		t.Errorf("round trip mismatch:\ngot  %#v\nwant %#v", roundTripped, m)
	}
}
