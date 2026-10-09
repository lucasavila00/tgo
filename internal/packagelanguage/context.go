package packagelanguage

import (
	"errors"
	"fmt"
	"go/build/constraint"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"tgo/internal/outputname"
)

// DefaultContext returns the active process build target for source tools.
func DefaultContext() (Context, error) {
	goos := environmentDefault("GOOS", runtime.GOOS)
	goarch := environmentDefault("GOARCH", runtime.GOARCH)
	tags, err := BuildTagsFromGoFlags(os.Getenv("GOFLAGS"))
	if err != nil {
		return Context{}, err
	}
	active := defaultTags(goos, goarch, tags)
	cgo := os.Getenv("CGO_ENABLED") != "0" && goarch != "wasm"
	return Context{
		CgoEnabled: cgo,
		MatchFile: func(path string, _ Language) (bool, error) {
			return matchFile(path, goos, goarch, active)
		},
	}, nil
}

// BuildTagsFromGoFlags returns each user build tag from GOFLAGS.
func BuildTagsFromGoFlags(flags string) ([]string, error) {
	fields, err := splitGoFlags(flags)
	if err != nil {
		return nil, err
	}
	var tags []string
	for index := 0; index < len(fields); index++ {
		field := fields[index]
		value, found := strings.CutPrefix(field, "-tags=")
		if !found && field == "-tags" {
			if index+1 >= len(fields) {
				return nil, errors.New("-tags needs a value")
			}
			index++
			value = fields[index]
			found = true
		}
		if found {
			tags = strings.FieldsFunc(value, func(character rune) bool {
				return character == ',' || character == ' '
			})
		}
	}
	return tags, nil
}

func splitGoFlags(value string) ([]string, error) {
	var fields []string
	for len(value) > 0 {
		value = strings.TrimLeft(value, " \t\n\r")
		if value == "" {
			break
		}
		if value[0] == '\'' || value[0] == '"' {
			quote := value[0]
			end := strings.IndexByte(value[1:], quote)
			if end < 0 {
				return nil, fmt.Errorf("unterminated %c string", quote)
			}
			fields = append(fields, value[1:end+1])
			value = value[end+2:]
			continue
		}
		end := strings.IndexAny(value, " \t\n\r")
		if end < 0 {
			fields = append(fields, value)
			break
		}
		fields = append(fields, value[:end])
		value = value[end:]
	}
	return fields, nil
}

func matchFile(
	path string,
	goos string,
	goarch string,
	tags map[string]bool,
) (bool, error) {
	name := filepath.Base(path)
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
		!outputname.MatchesTarget(name, goos, goarch) {
		return false, nil
	}
	expression, err := fileConstraint(path)
	if err != nil || expression == nil {
		return err == nil, err
	}
	return expression.Eval(func(tag string) bool { return tags[tag] }), nil
}

func fileConstraint(path string) (constraint.Expr, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var expression constraint.Expr
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			break
		}
		if !constraint.IsGoBuild(line) && !constraint.IsPlusBuild(line) {
			continue
		}
		item, err := constraint.Parse(line)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if expression == nil {
			expression = item
		} else {
			expression = &constraint.AndExpr{X: expression, Y: item}
		}
	}
	return expression, nil
}

func defaultTags(goos, goarch string, user []string) map[string]bool {
	tags := map[string]bool{goos: true, goarch: true, "gc": true}
	if os.Getenv("CGO_ENABLED") != "0" && goarch != "wasm" {
		tags["cgo"] = true
	}
	if goos == "android" {
		tags["linux"] = true
	}
	if goos == "illumos" {
		tags["solaris"] = true
	}
	if goos == "ios" {
		tags["darwin"] = true
	}
	if unixTarget(goos) {
		tags["unix"] = true
	}
	for _, tag := range releaseTags() {
		tags[tag] = true
	}
	for _, tag := range user {
		tags[tag] = true
	}
	return tags
}

func releaseTags() []string {
	version := runtime.Version()
	start := strings.Index(version, "go1.")
	if start < 0 {
		return nil
	}
	minorText := version[start+len("go1."):]
	end := strings.IndexFunc(minorText, func(character rune) bool {
		return character < '0' || character > '9'
	})
	if end >= 0 {
		minorText = minorText[:end]
	}
	minor, err := strconv.Atoi(minorText)
	if err != nil {
		return nil
	}
	tags := make([]string, 0, minor)
	for number := 1; number <= minor; number++ {
		tags = append(tags, "go1."+strconv.Itoa(number))
	}
	return tags
}

func unixTarget(goos string) bool {
	switch goos {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd",
		"illumos", "ios", "linux", "netbsd", "openbsd", "solaris":
		return true
	default:
		return false
	}
}

func environmentDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
