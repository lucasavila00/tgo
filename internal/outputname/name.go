// Package outputname maps tgo source names to generated Go names.
package outputname

import (
	"go/build"
	"io"
	"path/filepath"
	"strings"
)

// Path returns the generated Go path for one tgo source path.
func Path(source string) string {
	directory := filepath.Dir(source)
	name := strings.TrimSuffix(filepath.Base(source), ".tgo")
	test := strings.HasSuffix(name, "_test")
	if test {
		name = strings.TrimSuffix(name, "_test")
	}
	parts := strings.Split(name, "_")
	suffix := len(parts)
	lastOS, lastArch := targetSuffixKind(parts[len(parts)-1])
	previousOS := false
	if len(parts) > 2 {
		previousOS, _ = targetSuffixKind(parts[len(parts)-2])
	}
	if len(parts) > 2 && previousOS && lastArch {
		suffix = len(parts) - 2
	} else if len(parts) > 1 && (lastOS || lastArch) {
		suffix = len(parts) - 1
	}
	testSuffix := ""
	if test {
		testSuffix = "_test"
	}
	if suffix == len(parts) {
		return filepath.Join(directory, name+"_tgo"+testSuffix+".go")
	}
	prefix := strings.Join(parts[:suffix], "_")
	target := strings.Join(parts[suffix:], "_")
	return filepath.Join(directory, prefix+"_tgo_"+target+testSuffix+".go")
}

// Matches reports whether a generated path belongs to a source basename.
func Matches(source string, generated string) bool {
	if filepath.Base(source) != source || filepath.Ext(source) != ".tgo" {
		return false
	}
	return filepath.Base(Path(source)) == filepath.Base(generated)
}

// Reserved reports whether a path is in the TGo output namespace.
func Reserved(path string) bool {
	name := filepath.Base(path)
	if !strings.HasSuffix(name, ".go") {
		return false
	}
	stem := strings.TrimSuffix(name, ".go")
	stem = strings.TrimSuffix(stem, "_test")
	if strings.HasSuffix(stem, "_tgo") {
		return true
	}
	marker := strings.LastIndex(stem, "_tgo_")
	if marker < 0 {
		return false
	}
	parts := strings.Split(stem[marker+len("_tgo_"):], "_")
	switch len(parts) {
	case 1:
		isOS, isArch := targetSuffixKind(parts[0])
		return isOS || isArch
	case 2:
		isOS, _ := targetSuffixKind(parts[0])
		_, isArch := targetSuffixKind(parts[1])
		return isOS && isArch
	default:
		return false
	}
}

func targetSuffixKind(word string) (bool, bool) {
	const noOS = "tgo_unknown_os"
	const noArch = "tgo_unknown_arch"
	if matchTargetWord(word, noOS, noArch) {
		return false, false
	}
	return matchTargetWord(word, word, noArch), matchTargetWord(word, noOS, word)
}

func matchTargetWord(word string, goos string, goarch string) bool {
	if word == "tgo_unknown_os" || word == "tgo_unknown_arch" {
		return false
	}
	context := build.Default
	context.GOOS = goos
	context.GOARCH = goarch
	context.BuildTags = nil
	context.ToolTags = nil
	context.ReleaseTags = nil
	context.OpenFile = func(string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("")), nil
	}
	match, err := context.MatchFile(".", "source_"+word+".s")
	return err == nil && match
}
