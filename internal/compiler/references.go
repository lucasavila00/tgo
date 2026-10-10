package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"tgo/pkg/syntax"

	"golang.org/x/tools/go/ast/astutil"
)

type enumNameReservation struct {
	allowed token.Pos
	owner   string
}

// checkGeneratedEnumNameCollisions reserves the package-level enum ABI.
func (p *packageUnit) checkGeneratedEnumNameCollisions() {
	reserved := p.generatedEnumNameReservations()
	p.checkTGoEnumNameCollisions(reserved)
	p.checkGoEnumNameCollisions(reserved)
}

func (p *packageUnit) generatedEnumNameReservations() map[string]enumNameReservation {
	reserved := make(map[string]enumNameReservation)
	for _, source := range p.Sources {
		for _, declaration := range source.Tree.Declarations {
			node, ok := syntax.EnumDeclarationOf(declaration)
			if !ok || node == nil {
				continue
			}
			p.reserveEnumDeclarationNames(reserved, node)
		}
	}
	return reserved
}

func (p *packageUnit) reserveEnumDeclarationNames(
	reserved map[string]enumNameReservation,
	node *syntax.EnumDeclaration,
) {
	owner := "enum " + node.Name.Name
	p.reserveEnumName(reserved, node.Name.Name, node.Name.Start, owner, node.Name.Start)
	p.reserveEnumName(reserved, node.Name.Name+"Tag", token.NoPos, owner, node.Name.Start)
	for _, item := range node.Variants {
		p.reserveEnumVariantNames(reserved, node.Name.Name, owner, item)
	}
}

func (p *packageUnit) reserveEnumVariantNames(
	reserved map[string]enumNameReservation,
	enum string,
	owner string,
	item *syntax.EnumVariant,
) {
	p.reserveEnumName(reserved, enum+item.Name.Name, token.NoPos, owner, item.Name.Start)
	p.reserveEnumName(reserved, enum+"Tag"+item.Name.Name, token.NoPos, owner, item.Name.Start)
	p.reserveEnumName(
		reserved,
		enumConstructorName(enum, item.Name.Name),
		token.NoPos,
		owner,
		item.Name.Start,
	)
	if len(item.Fields) > 0 {
		p.reserveEnumName(
			reserved,
			enumCarrierName(enum, item.Name.Name),
			token.NoPos,
			owner,
			item.Name.Start,
		)
	}
	for _, itemField := range item.Fields {
		if itemField.Default == nil {
			continue
		}
		for _, fieldName := range itemField.Field.Names {
			p.reserveEnumName(
				reserved,
				"TgoDefault"+enum+item.Name.Name+fieldName.Name,
				token.NoPos,
				owner,
				fieldName.Start,
			)
		}
	}
}

func (p *packageUnit) reserveEnumName(
	reserved map[string]enumNameReservation,
	name string,
	allowed token.Pos,
	owner string,
	at token.Pos,
) {
	if previous, exists := reserved[name]; exists {
		if previous.allowed != allowed || previous.owner != owner {
			p.failAt(at, "generated enum name %s conflicts with %s", name, previous.owner)
		}
		return
	}
	reserved[name] = enumNameReservation{allowed: allowed, owner: owner}
}

func (p *packageUnit) checkTGoEnumNameCollisions(
	reserved map[string]enumNameReservation,
) {
	for _, source := range p.Sources {
		for _, declaration := range source.Tree.Declarations {
			for _, declared := range sourceDeclarationNames(declaration) {
				reservation, exists := reserved[declared.Name]
				if exists && declared.Position != reservation.allowed {
					p.failAt(
						declared.Position,
						"name %s is reserved by %s",
						declared.Name,
						reservation.owner,
					)
				}
			}
		}
	}
}

func (p *packageUnit) checkGoEnumNameCollisions(
	reserved map[string]enumNameReservation,
) {
	tgoFiles := make(map[*ast.File]bool)
	for _, source := range p.Sources {
		tgoFiles[source.File] = true
	}
	for _, file := range p.Files {
		if tgoFiles[file] {
			continue
		}
		for _, declaration := range file.Decls {
			for _, declared := range goDeclarationNames(declaration) {
				if reservation, exists := reserved[declared.Name]; exists {
					p.failAt(
						declared.Position,
						"name %s is reserved by %s",
						declared.Name,
						reservation.owner,
					)
				}
			}
		}
	}
}

type namedPosition struct {
	Name     string
	Position token.Pos
}

func sourceDeclarationNames(declaration *syntax.Declaration) []namedPosition {
	var result []namedPosition
	if node := syntax.GeneralDeclarationOf(declaration); node != nil {
		for _, specification := range node.Specs {
			if value := syntax.ValueSpecificationOf(specification); value != nil {
				for _, name := range value.Names {
					result = append(result, namedPosition{name.Name, name.Start})
				}
			}
			if value := syntax.TypeSpecificationOf(specification); value != nil {
				result = append(result, namedPosition{value.Name.Name, value.Name.Start})
			}
		}
	}
	if node := syntax.FunctionDeclarationValueOf(declaration); node != nil && node.Receiver == nil {
		result = append(result, namedPosition{node.Name.Name, node.Name.Start})
	}
	if node, ok := syntax.EnumDeclarationOf(declaration); ok && node != nil {
		result = append(result, namedPosition{node.Name.Name, node.Name.Start})
	}
	if node, ok := syntax.StructDeclarationOf(declaration); ok && node != nil {
		result = append(result, namedPosition{node.Name.Name, node.Name.Start})
	}
	return result
}

func goDeclarationNames(declaration ast.Decl) []namedPosition {
	var result []namedPosition
	switch node := declaration.(type) {
	case *ast.GenDecl:
		for _, specification := range node.Specs {
			switch item := specification.(type) {
			case *ast.TypeSpec:
				result = append(result, namedPosition{item.Name.Name, item.Name.Pos()})
			case *ast.ValueSpec:
				for _, name := range item.Names {
					result = append(result, namedPosition{name.Name, name.Pos()})
				}
			}
		}
	case *ast.FuncDecl:
		if node.Recv == nil {
			result = append(result, namedPosition{node.Name.Name, node.Name.Pos()})
		}
	}
	return result
}

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
	if imported := surfaceImport(expression, p.info); imported != nil {
		for _, specification := range file.Imports {
			if importPackageName(specification, p.info) == imported {
				p.erasedImports[specification] = file
				return
			}
		}
		return
	}
	for _, specification := range file.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err == nil && path == surface.Path() &&
			specification.Name != nil && specification.Name.Name == "." {
			p.erasedImports[specification] = file
			return
		}
	}
}

// surfaceImport gets the qualified import named by one source type expression.
func surfaceImport(expression ast.Expr, info *types.Info) *types.PkgName {
	switch expression := expression.(type) {
	case *ast.SelectorExpr:
		identifier, ok := expression.X.(*ast.Ident)
		if !ok {
			return nil
		}
		imported, _ := info.Uses[identifier].(*types.PkgName)
		return imported
	case *ast.IndexExpr:
		return surfaceImport(expression.X, info)
	case *ast.IndexListExpr:
		return surfaceImport(expression.X, info)
	case *ast.ParenExpr:
		return surfaceImport(expression.X, info)
	}
	return nil
}

// importPackageName gets the object declared by one import specification.
func importPackageName(
	specification *ast.ImportSpec,
	info *types.Info,
) *types.PkgName {
	if specification.Name == nil {
		imported, _ := info.Implicits[specification].(*types.PkgName)
		return imported
	}
	imported, _ := info.Defs[specification.Name].(*types.PkgName)
	return imported
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
	for specification, file := range p.erasedImports {
		if specification.Name != nil && specification.Name.Name == "_" {
			continue
		}
		if p.importUsed(file, specification) {
			continue
		}
		specification.Name = ast.NewIdent("_")
		changed = true
	}
	return changed
}

// importUsed reports whether lowered syntax still refers to one import.
func (p *packageUnit) importUsed(
	file *ast.File,
	specification *ast.ImportSpec,
) bool {
	path, err := strconv.Unquote(specification.Path.Value)
	if err != nil {
		return true
	}
	if specification.Name != nil && specification.Name.Name == "." {
		return p.dotImportUsed(file, path)
	}
	imported := importPackageName(specification, p.info)
	for _, object := range p.info.Uses {
		if object == imported {
			return true
		}
	}
	return false
}

// dotImportUsed reports whether one file still has an unqualified package use.
func (p *packageUnit) dotImportUsed(file *ast.File, path string) bool {
	used := false
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		identifier, ok := cursor.Node().(*ast.Ident)
		if !ok {
			return true
		}
		if selector, ok := cursor.Parent().(*ast.SelectorExpr); ok &&
			selector.Sel == identifier {
			return true
		}
		object := p.info.Uses[identifier]
		if object == nil || object.Pkg() == nil || object.Pkg().Path() != path ||
			object.Parent() != object.Pkg().Scope() {
			return true
		}
		used = true
		return false
	}, nil)
	return used
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
				names = append(names, "new")
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
