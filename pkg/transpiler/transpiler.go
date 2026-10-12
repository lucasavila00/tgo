package transpiler

import "github.com/lucasavila00/tgo/internal/driver"

// Compile translates the module's TGo files into Go files in the same module.
// On failure, it restores previous generated files and panics.
func Compile(dir string) { driver.Compile(dir) }
