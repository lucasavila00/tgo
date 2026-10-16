package driver

import "github.com/lucasavila00/tgo/internal/sourcefile"

// generatedName keeps Go build and test suffixes after the output marker.
func generatedName(source string) string {
	return sourcefile.GeneratedName(source)
}
