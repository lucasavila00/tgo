package transpiler

import (
	"github.com/lucasavila00/tgo/internal/driver"
	"github.com/lucasavila00/tgo/internal/lowering"
	"golang.org/x/tools/go/packages"
)

// Site identifies an expression in the original source. Offsets are bytes.
type Site = lowering.Site

// Source contains the original syntax and its type information.
type Source struct {
	Package *packages.Package
	Sites   []Site
	source  lowering.Source
}

// After receives references to the evaluated expression results.
type After = lowering.After

// Load reads one Go package. It does not change its syntax tree.
func Load(dir, pattern string) (*Source, error) {
	source, err := lowering.Load(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &Source{Package: source.Package, Sites: source.Sites, source: *source}, nil
}

// InsertAfter builds source files with statements after one expression.
// The callback uses the expression's evaluated results in source order.
func (s *Source) InsertAfter(site Site, after After) (map[string][]byte, error) {
	return s.source.InsertAfter(site, after)
}

// Compile translates the module's TGo files into Go files in the same module.
// Invalid source and compiler failures panic.
func Compile(dir string) { driver.Compile(dir) }
