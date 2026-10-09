package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

// generatedReference records the object that one inserted name must use.
type generatedReference struct {
	Name     *ast.Ident
	At       token.Pos
	Path     string
	Universe bool
}

// generatedObject makes a local or qualified reference and records its owner.
func (p *packageUnit) generatedObject(
	prefix string,
	path string,
	name string,
	at token.Pos,
) ast.Expr {
	identifier := ast.NewIdent(name)
	p.references = append(p.references, generatedReference{
		Name: identifier,
		At:   at,
		Path: path,
	})
	if prefix == "" {
		return identifier
	}
	return &ast.SelectorExpr{X: ast.NewIdent(prefix), Sel: identifier}
}

// generatedUniverse makes a reference to one predeclared Go object.
func (p *packageUnit) generatedUniverse(name string, at token.Pos) *ast.Ident {
	identifier := ast.NewIdent(name)
	p.references = append(p.references, generatedReference{
		Name:     identifier,
		At:       at,
		Universe: true,
	})
	return identifier
}

// ownerQualifier gets or adds a file import for the canonical type owner.
func (p *packageUnit) ownerQualifier(file *ast.File, owner *types.Package) string {
	if owner == nil || owner.Path() == p.Path {
		return ""
	}
	for _, specification := range file.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil || path != owner.Path() {
			continue
		}
		if specification.Name == nil {
			if imported, ok := p.info.Implicits[specification].(*types.PkgName); ok {
				return imported.Name()
			}
			return owner.Name()
		}
		switch specification.Name.Name {
		case ".":
			return ""
		case "_":
			alias := freshASTIdentifier(file, owner.Name())
			if alias == owner.Name() {
				specification.Name = nil
			} else {
				specification.Name = ast.NewIdent(alias)
			}
			return alias
		default:
			return specification.Name.Name
		}
	}
	alias := freshASTIdentifier(file, owner.Name())
	if alias == owner.Name() {
		astutil.AddImport(p.fs, file, owner.Path())
	} else {
		astutil.AddNamedImport(p.fs, file, alias, owner.Path())
	}
	return alias
}

// markErasedOwnerImport records an alias-owner import removed by lowering.
func (p *packageUnit) markErasedOwnerImport(
	file *ast.File,
	expression ast.Expr,
	owner *types.Package,
) {
	surface := surfaceTypePackage(expression, p.info)
	if surface == nil || owner == nil || surface.Path() == owner.Path() {
		return
	}
	for _, specification := range file.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err == nil && path == surface.Path() {
			p.erasedImports[specification] = true
			return
		}
	}
}

// surfaceTypePackage gets the package named by the source type expression.
func surfaceTypePackage(expression ast.Expr, info *types.Info) *types.Package {
	switch expression := expression.(type) {
	case *ast.Ident:
		if object := info.Uses[expression]; object != nil {
			return object.Pkg()
		}
	case *ast.SelectorExpr:
		if object := info.Uses[expression.Sel]; object != nil {
			return object.Pkg()
		}
	case *ast.IndexExpr:
		return surfaceTypePackage(expression.X, info)
	case *ast.IndexListExpr:
		return surfaceTypePackage(expression.X, info)
	case *ast.ParenExpr:
		return surfaceTypePackage(expression.X, info)
	}
	return nil
}

// blankUnusedErasedImports preserves imports whose only type use was lowered.
func (p *packageUnit) blankUnusedErasedImports() bool {
	changed := false
	for specification := range p.erasedImports {
		if specification.Name != nil && specification.Name.Name == "_" {
			continue
		}
		if p.importUsed(specification) {
			continue
		}
		specification.Name = ast.NewIdent("_")
		changed = true
	}
	return changed
}

// importUsed reports whether lowered syntax still refers to one import.
func (p *packageUnit) importUsed(specification *ast.ImportSpec) bool {
	path, err := strconv.Unquote(specification.Path.Value)
	if err != nil {
		return true
	}
	if specification.Name != nil && specification.Name.Name == "." {
		for _, object := range p.info.Uses {
			if object != nil && object.Pkg() != nil && object.Pkg().Path() == path {
				return true
			}
		}
		return false
	}
	var imported types.Object
	if specification.Name == nil {
		imported = p.info.Implicits[specification]
	} else {
		imported = p.info.Defs[specification.Name]
	}
	for _, object := range p.info.Uses {
		if object == imported {
			return true
		}
	}
	return false
}

// validateGeneratedReferences rejects a source name that captures inserted code.
func (p *packageUnit) validateGeneratedReferences() {
	for _, reference := range p.references {
		actual := p.info.Uses[reference.Name]
		expected := p.expectedObject(reference)
		if expected == nil || actual != expected {
			p.failAt(reference.At, "generated name %s is shadowed", reference.Name.Name)
		}
	}
}

// expectedObject finds one intended object in the current type-check result.
func (p *packageUnit) expectedObject(reference generatedReference) types.Object {
	if reference.Universe {
		return types.Universe.Lookup(reference.Name.Name)
	}
	owner := packageByPath(p.typed, reference.Path, make(map[*types.Package]bool))
	if owner == nil {
		return nil
	}
	return owner.Scope().Lookup(reference.Name.Name)
}

// packageByPath finds a package in one type-check import graph.
func packageByPath(
	root *types.Package,
	path string,
	seen map[*types.Package]bool,
) *types.Package {
	if root == nil || seen[root] {
		return nil
	}
	seen[root] = true
	if root.Path() == path {
		return root
	}
	for _, imported := range root.Imports() {
		if found := packageByPath(imported, path, seen); found != nil {
			return found
		}
	}
	return nil
}

// checkGeneratedPredeclaredNames keeps generated declarations on Go built-ins.
func (p *packageUnit) checkGeneratedPredeclaredNames() {
	checked := make(map[string]bool)
	for _, source := range p.Sources {
		for _, declaration := range source.Models {
			names := []string{"any", "error", "nil", "string"}
			if len(declaration.Variants) > 0 {
				names = append(names, enumTagType(len(declaration.Variants)))
			}
			for _, name := range names {
				key := source.Name + "\x00" + name
				if checked[key] {
					continue
				}
				checked[key] = true
				at := modelName(source.File, declaration.Name)
				if p.objectAt(at.Pos(), name) != types.Universe.Lookup(name) {
					message := "predeclared name " + name + " is shadowed"
					p.errors = append(p.errors, fmt.Errorf(
						"%s:%d:%d: %s",
						source.Name,
						declaration.Line,
						declaration.Column,
						message,
					))
				}
			}
		}
	}
}

// objectAt resolves one name in the lexical scope at a source position.
func (p *packageUnit) objectAt(position token.Pos, name string) types.Object {
	scope := p.typed.Scope().Innermost(position)
	if scope == nil {
		return nil
	}
	_, object := scope.LookupParent(name, position)
	return object
}

// modelName gets the declared type name node for one source model.
func modelName(file *ast.File, name string) ast.Node {
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, item := range general.Specs {
			specification, ok := item.(*ast.TypeSpec)
			if ok && specification.Name.Name == name {
				return specification.Name
			}
		}
	}
	return file.Name
}
