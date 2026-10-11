// Package transpiler emits Go source from TGo modules.
package transpiler

import (
	"github.com/lucasavila00/tgo/transpiler/internal/driver"
	"github.com/lucasavila00/tgo/transpiler/internal/lowering"
)

// Site identifies an expression in the original source. Offsets are bytes.
type Site = lowering.Site

// Source contains the original syntax and its type information.
type Source = lowering.Source

// After receives references to the evaluated expression results.
type After = lowering.After

// Load reads one Go package. It does not change its syntax tree.
func Load(dir, pattern string) (*Source, error) { return lowering.Load(dir, pattern) }

// Compile emits a Go module from the TGo files in a module directory.
// Invalid source and compiler failures panic.
func Compile(dir string) map[string][]byte { return driver.Compile(dir) }
