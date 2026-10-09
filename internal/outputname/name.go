// Package outputname maps tgo source names to generated Go names.
package outputname

import (
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
	return knownTargetOS(word), knownTargetArch(word)
}

// MatchesTarget reports whether target suffixes allow a source file name.
func MatchesTarget(name, goos, goarch string) bool {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	stem = strings.TrimSuffix(stem, "_test")
	parts := strings.Split(stem, "_")
	if len(parts) < 2 {
		return true
	}
	lastOS, lastArch := targetSuffixKind(parts[len(parts)-1])
	if len(parts) > 2 {
		previousOS, _ := targetSuffixKind(parts[len(parts)-2])
		if previousOS && lastArch {
			return parts[len(parts)-2] == goos && parts[len(parts)-1] == goarch
		}
	}
	if lastOS {
		return parts[len(parts)-1] == goos
	}
	if lastArch {
		return parts[len(parts)-1] == goarch
	}
	return true
}

func knownTargetOS(word string) bool {
	switch word {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd",
		"illumos", "ios", "js", "linux", "netbsd", "openbsd", "plan9",
		"solaris", "wasip1", "windows", "zos":
		return true
	default:
		return false
	}
}

func knownTargetArch(word string) bool {
	switch word {
	case "386", "amd64", "arm", "arm64", "loong64", "mips", "mips64",
		"mips64le", "mipsle", "ppc64", "ppc64le", "riscv64", "s390x", "wasm":
		return true
	default:
		return false
	}
}
