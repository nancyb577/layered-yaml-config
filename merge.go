package yamlconf

import (
	"fmt"
	"sort"
	"strings"
)

// Merge combines override on top of base and returns a new map. Nested
// maps are merged recursively; any other value in override replaces the
// base value entirely, including replacing a map with a scalar.
func Merge(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		if baseVal, ok := result[k]; ok {
			baseMap, baseIsMap := baseVal.(map[string]interface{})
			overrideMap, overrideIsMap := v.(map[string]interface{})
			if baseIsMap && overrideIsMap {
				result[k] = Merge(baseMap, overrideMap)
				continue
			}
		}
		result[k] = v
	}
	return result
}

// Dump renders m back into the same subset of YAML that Parse accepts,
// with map keys sorted so the output is stable across runs.
func Dump(m map[string]interface{}) string {
	var b strings.Builder
	dumpMap(&b, m, 0)
	return b.String()
}

func dumpMap(b *strings.Builder, m map[string]interface{}, depth int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	prefix := strings.Repeat("  ", depth)
	for _, k := range keys {
		v := m[k]
		switch val := v.(type) {
		case map[string]interface{}:
			fmt.Fprintf(b, "%s%s:\n", prefix, k)
			if len(val) > 0 {
				dumpMap(b, val, depth+1)
			}
		case []interface{}:
			fmt.Fprintf(b, "%s%s:\n", prefix, k)
			dumpList(b, val, depth+1)
		default:
			fmt.Fprintf(b, "%s%s: %s\n", prefix, k, formatScalar(v))
		}
	}
}

func dumpList(b *strings.Builder, items []interface{}, depth int) {
	prefix := strings.Repeat("  ", depth)
	for _, item := range items {
		fmt.Fprintf(b, "%s- %s\n", prefix, formatScalar(item))
	}
}

func formatScalar(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case string:
		if val == "" || needsQuoting(val) {
			return fmt.Sprintf("%q", val)
		}
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// needsQuoting reports whether a plain scalar would round-trip through
// Parse ambiguously (e.g. as a bool or null) and must be quoted instead.
func needsQuoting(s string) bool {
	switch s {
	case "true", "false", "null", "~":
		return true
	}
	return strings.ContainsAny(s, ":#'\"") || strings.TrimSpace(s) != s
}
