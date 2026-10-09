package packagelanguage

import (
	"errors"
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DefaultContext returns the active process build target for source tools.
func DefaultContext() (Context, error) {
	tags, err := BuildTagsFromGoFlags(os.Getenv("GOFLAGS"))
	if err != nil {
		return Context{}, err
	}
	buildContext := build.Default
	buildContext.BuildTags = append(
		append([]string(nil), buildContext.BuildTags...),
		tags...,
	)
	return ContextFromBuild(&buildContext), nil
}

// ContextFromBuild returns a package language context for one Go build context.
func ContextFromBuild(buildContext *build.Context) Context {
	return Context{
		CgoEnabled: buildContext.CgoEnabled,
		MatchFile: func(path string, language Language) (bool, error) {
			return MatchFile(buildContext, path, language)
		},
	}
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

// MatchFile applies a Go build context to one Go or TGo source file.
func MatchFile(
	buildContext *build.Context,
	path string,
	language Language,
) (bool, error) {
	directory := filepath.Dir(path)
	name := filepath.Base(path)
	if language == Go {
		return buildContext.MatchFile(directory, name)
	}
	fakeName := strings.TrimSuffix(name, ".tgo") + ".s"
	fakePath := filepath.Clean(filepath.Join(directory, fakeName))
	realPath := filepath.Clean(path)
	fileContext := *buildContext
	fileContext.OpenFile = func(requested string) (io.ReadCloser, error) {
		if filepath.Clean(requested) == fakePath {
			return os.Open(realPath)
		}
		return os.Open(requested)
	}
	match, err := fileContext.MatchFile(directory, fakeName)
	if err == nil {
		return match, nil
	}
	message := err.Error()
	if detail, ok := strings.CutPrefix(message, fakeName); ok {
		message = path + detail
	}
	return false, errors.New(message)
}
