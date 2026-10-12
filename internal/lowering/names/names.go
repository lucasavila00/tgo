// Package names allocates generated identifiers without source name collisions.
// One allocator spans all files in a rewrite, so the output order stays stable.
package names

import (
	"fmt"
	"go/ast"
)

// Allocator keeps source names and previously generated names.
type Allocator struct {
	used map[string]bool
	next int
}

// Reserve records names that generated identifiers must not use.
func (a *Allocator) Reserve(names ...string) {
	if a.used == nil {
		a.used = make(map[string]bool)
	}
	for _, name := range names {
		a.used[name] = true
	}
}

// ReserveSyntax records all source identifiers before generation starts.
func (a *Allocator) ReserveSyntax(files []*ast.File) {
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok {
				a.Reserve(identifier.Name)
			}
			return true
		})
	}
}

// Fresh adds the source name to a fresh generated identifier.
func (a *Allocator) Fresh(sourceName string) *ast.Ident {
	for {
		a.next++
		name := fmt.Sprintf("tgo%d", a.next)
		if sourceName != "" {
			name = sourceName + "_" + name
		}
		if !a.used[name] {
			a.Reserve(name)
			return ast.NewIdent(name)
		}
	}
}
