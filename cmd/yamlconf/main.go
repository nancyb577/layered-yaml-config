// Command yamlconf reads and merges layered YAML config files from the
// command line.
package main

import (
	"fmt"
	"os"
	"strings"

	yamlconf "github.com/nancyb577/layered-yaml-config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "get":
		err = runGet(os.Args[2:])
	case "merge":
		err = runMerge(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "yamlconf:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  yamlconf get <file> <dotted.path>
  yamlconf merge <file> [<file> ...]`)
}

func runGet(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("get needs a file and a dotted path")
	}
	cfg, err := yamlconf.ParseFile(args[0])
	if err != nil {
		return err
	}
	cfg, err = yamlconf.ExpandEnv(cfg)
	if err != nil {
		return err
	}
	val, ok := lookup(cfg, strings.Split(args[1], "."))
	if !ok {
		return fmt.Errorf("path %q not found", args[1])
	}
	fmt.Println(val)
	return nil
}

func runMerge(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("merge needs at least one file")
	}
	result, err := yamlconf.ParseFile(args[0])
	if err != nil {
		return err
	}
	for _, path := range args[1:] {
		layer, err := yamlconf.ParseFile(path)
		if err != nil {
			return err
		}
		result = yamlconf.Merge(result, layer)
	}
	result, err = yamlconf.ExpandEnv(result)
	if err != nil {
		return err
	}
	fmt.Print(yamlconf.Dump(result))
	return nil
}

func lookup(m map[string]interface{}, path []string) (interface{}, bool) {
	var current interface{} = m
	for _, part := range path {
		asMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = asMap[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
