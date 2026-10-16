package sourcefile

import (
	"path/filepath"
	"strings"
)

// GeneratedName keeps Go build and test suffixes after the output marker.
func GeneratedName(source string) string {
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

// targetSuffixKind reports whether a filename suffix identifies a Go operating system or architecture.
func targetSuffixKind(word string) (bool, bool) {
	return knownTargetOS(word), knownTargetArch(word)
}

// knownTargetOS reports whether a word is a recognized Go operating-system suffix.
func knownTargetOS(word string) bool {
	switch word {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd",
		"illumos", "ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9",
		"solaris", "wasip1", "windows", "zos":
		return true
	default:
		return false
	}
}

// knownTargetArch reports whether a word is a recognized Go architecture suffix.
func knownTargetArch(word string) bool {
	switch word {
	case "386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "loong64", "mips", "mips64",
		"mips64le", "mips64p32", "mips64p32le", "mipsle", "ppc", "ppc64", "ppc64le", "riscv", "riscv64", "s390",
		"s390x", "sparc", "sparc64", "wasm":
		return true
	default:
		return false
	}
}

// InputName returns the TGo input path for a generated Go filename.
// An empty result means the filename does not follow the output naming rule.
func InputName(output string) string {
	if !strings.HasSuffix(output, ".go") {
		return ""
	}
	name := strings.TrimSuffix(filepath.Base(output), ".go")
	marker := strings.LastIndex(name, "_tgo")
	if marker < 0 {
		return ""
	}
	input := filepath.Join(filepath.Dir(output), name[:marker]+name[marker+4:]+".tgo")
	if GeneratedName(input) != output {
		return ""
	}
	return input
}
