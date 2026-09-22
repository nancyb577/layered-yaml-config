package yamlconf

import (
	"fmt"
	"os"
	"regexp"
)

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// Expand walks m and replaces every ${VAR} reference in a string scalar
// with the value lookup returns for VAR. Map keys, and non-string scalars
// such as numbers and bools, are left untouched. It returns a new map;
// the input is not modified.
//
// If lookup reports a variable as unset, Expand returns an error naming
// the variable and the dotted path of the value it appeared in, rather
// than silently substituting an empty string.
func Expand(m map[string]interface{}, lookup func(string) (string, bool)) (map[string]interface{}, error) {
	return expandMap(m, lookup, "")
}

// ExpandEnv is Expand using the process environment via os.LookupEnv.
func ExpandEnv(m map[string]interface{}) (map[string]interface{}, error) {
	return Expand(m, os.LookupEnv)
}

func expandMap(m map[string]interface{}, lookup func(string) (string, bool), path string) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		childPath := k
		if path != "" {
			childPath = path + "." + k
		}
		expanded, err := expandValue(v, lookup, childPath)
		if err != nil {
			return nil, err
		}
		result[k] = expanded
	}
	return result, nil
}

func expandValue(v interface{}, lookup func(string) (string, bool), path string) (interface{}, error) {
	switch val := v.(type) {
	case string:
		return expandString(val, lookup, path)
	case map[string]interface{}:
		return expandMap(val, lookup, path)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			expanded, err := expandValue(item, lookup, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			result[i] = expanded
		}
		return result, nil
	default:
		return v, nil
	}
}

func expandString(s string, lookup func(string) (string, bool), path string) (string, error) {
	var err error
	result := envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		if err != nil {
			return match
		}
		name := envVarPattern.FindStringSubmatch(match)[1]
		val, ok := lookup(name)
		if !ok {
			err = fmt.Errorf("%s: environment variable %q is not set", path, name)
			return match
		}
		return val
	})
	if err != nil {
		return "", err
	}
	return result, nil
}
