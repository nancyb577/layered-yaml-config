package yamlconf

import (
	"strings"
	"testing"
)

func lookupFrom(env map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := env[name]
		return v, ok
	}
}

func TestExpandReplacesVariable(t *testing.T) {
	m := map[string]interface{}{
		"host": "${DB_HOST}",
	}
	got, err := Expand(m, lookupFrom(map[string]string{"DB_HOST": "db.internal"}))
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if got["host"] != "db.internal" {
		t.Errorf("host = %#v, want %q", got["host"], "db.internal")
	}
}

func TestExpandMultipleVariablesInOneString(t *testing.T) {
	m := map[string]interface{}{
		"url": "https://${HOST}:${PORT}/",
	}
	got, err := Expand(m, lookupFrom(map[string]string{"HOST": "example.com", "PORT": "8080"}))
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if got["url"] != "https://example.com:8080/" {
		t.Errorf("url = %#v", got["url"])
	}
}

func TestExpandLeavesNonStringsAlone(t *testing.T) {
	m := map[string]interface{}{
		"port":    8080,
		"debug":   false,
		"nothing": nil,
	}
	got, err := Expand(m, lookupFrom(nil))
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if got["port"] != 8080 || got["debug"] != false || got["nothing"] != nil {
		t.Errorf("got = %#v", got)
	}
}

func TestExpandLeavesPlainDollarSignsAlone(t *testing.T) {
	m := map[string]interface{}{
		"price": "$5 off",
	}
	got, err := Expand(m, lookupFrom(nil))
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if got["price"] != "$5 off" {
		t.Errorf("price = %#v", got["price"])
	}
}

func TestExpandNestedMapsAndLists(t *testing.T) {
	m := map[string]interface{}{
		"service": map[string]interface{}{
			"origins": []interface{}{"${SCHEME}://a.example.com", "${SCHEME}://b.example.com"},
		},
	}
	got, err := Expand(m, lookupFrom(map[string]string{"SCHEME": "https"}))
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	service := got["service"].(map[string]interface{})
	origins := service["origins"].([]interface{})
	if origins[0] != "https://a.example.com" || origins[1] != "https://b.example.com" {
		t.Errorf("origins = %#v", origins)
	}
}

func TestExpandUndefinedVariableIsAnError(t *testing.T) {
	m := map[string]interface{}{
		"service": map[string]interface{}{
			"host": "${MISSING}",
		},
	}
	_, err := Expand(m, lookupFrom(nil))
	if err == nil {
		t.Fatal("want an error for an undefined variable, got nil")
	}
	if !strings.Contains(err.Error(), "MISSING") || !strings.Contains(err.Error(), "service.host") {
		t.Errorf("err = %v, want it to name the variable and the service.host path", err)
	}
}

func TestExpandDoesNotMutateInput(t *testing.T) {
	m := map[string]interface{}{
		"host": "${DB_HOST}",
	}
	_, err := Expand(m, lookupFrom(map[string]string{"DB_HOST": "db.internal"}))
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if m["host"] != "${DB_HOST}" {
		t.Errorf("input was mutated: %#v", m)
	}
}
