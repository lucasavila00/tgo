package shared

import (
	"go/token"
	"strconv"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
)

// enumImports removes qualifiers used only by a case replaced with default.
// A blank import retains package initialization when no other import remains.
func (r *Context) enumImports(output, source *ast.File) {
	removed := r.enumCaseImports(source)
	if len(removed) == 0 {
		return
	}
	r.retainEnumImports(output, removed)

	retainedPaths := make(map[string]bool)
	for _, declaration := range output.Decls {
		if group, ok := declaration.(*ast.GenDecl); ok && group.Tok == token.IMPORT {
			for _, specification := range group.Specs {
				imported := specification.(*ast.ImportSpec)
				if !removed[r.importObject(imported)] {
					path, _ := strconv.Unquote(imported.Path.Value)
					retainedPaths[path] = true
				}
			}
		}
	}
	output.Decls, output.Imports = r.copyEnumImports(output.Decls, removed, retainedPaths)
}

// enumCaseImports finds imported package bindings in source enum case names.
func (r *Context) enumCaseImports(file *ast.File) map[*types.PkgName]bool {
	imports := make(map[*types.PkgName]bool)
	dots := make(map[*types.Package]*types.PkgName)
	for _, specification := range file.Imports {
		imported := r.importObject(specification)
		if imported != nil && imported.Name() == "." {
			dots[imported.Imported()] = imported
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		clause, ok := node.(*ast.CaseClause)
		if !ok || len(r.input.Inventory.Checked.TypesInfo.EnumCases[clause]) == 0 {
			return true
		}
		for _, expression := range clause.List {
			ast.Inspect(expression, func(child ast.Node) bool {
				if identifier, ok := child.(*ast.Ident); ok {
					if imported, ok := r.input.Inventory.Checked.TypesInfo.Uses[identifier].(*types.PkgName); ok {
						imports[imported] = true
					}
					if object := r.input.Inventory.Checked.TypesInfo.Uses[identifier]; object != nil {
						if imported := dots[object.Pkg()]; imported != nil {
							imports[imported] = true
						}
					}
				}
				return true
			})
		}
		return true
	})
	return imports
}

// importObject returns the checked package binding for a written import.
func (r *Context) importObject(specification *ast.ImportSpec) *types.PkgName {
	object := r.input.Inventory.Checked.TypesInfo.Implicits[specification]
	if specification.Name != nil {
		object = r.input.Inventory.Checked.TypesInfo.Defs[specification.Name]
	}
	imported, _ := object.(*types.PkgName)
	return imported
}

// copyEnumImports copies import groups whose original qualifiers are no longer used.
func (r *Context) copyEnumImports(
	declarations []ast.Decl, removed map[*types.PkgName]bool, retainedPaths map[string]bool,
) ([]ast.Decl, []*ast.ImportSpec) {
	var output []ast.Decl
	var imports []*ast.ImportSpec
	for _, declaration := range declarations {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.IMPORT {
			output = append(output, declaration)
			continue
		}
		copyGroup := *group
		copyGroup.Specs = nil
		for _, specification := range group.Specs {
			imported := specification.(*ast.ImportSpec)
			if removed[r.importObject(imported)] {
				path, _ := strconv.Unquote(imported.Path.Value)
				if retainedPaths[path] {
					continue
				}
				copyImport := *imported
				copyImport.Name = ast.NewIdent("_")
				imported = &copyImport
			}
			copyGroup.Specs = append(copyGroup.Specs, imported)
			imports = append(imports, imported)
		}
		if len(copyGroup.Specs) != 0 {
			output = append(output, &copyGroup)
		}
	}
	return output, imports
}

// retainEnumImports leaves each package qualifier that still appears in generated syntax.
func (r *Context) retainEnumImports(output *ast.File, removed map[*types.PkgName]bool) {
	ast.Inspect(output, func(node ast.Node) bool {
		if name, ok := node.(*ast.Ident); ok {
			if imported, ok := r.input.Inventory.Checked.TypesInfo.Uses[name].(*types.PkgName); ok {
				delete(removed, imported)
			}
			if object := r.input.Inventory.Checked.TypesInfo.Uses[name]; object != nil {
				for imported := range removed {
					if imported.Name() == "." && object.Pkg() == imported.Imported() {
						delete(removed, imported)
					}
				}
			}
		}
		return true
	})
}
