// Command deaddecl reports unreachable package-level declarations.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type deadcodePackage struct {
	Funcs []deadcodeFunction
}

type deadcodeFunction struct {
	Name     string
	Position position
}

type position struct {
	File string
	Line int
	Col  int
}

type declaration struct {
	Name      string
	Kind      string
	Position  position
	Generated bool
}

type node struct {
	declaration declaration
	edges       map[string]bool
}

func main() {
	patterns := os.Args[1:]
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	deadFunctions := readDeadFunctions()
	root, loaded := loadPackages(patterns)
	nodes, roots := buildGraph(root, loaded, deadFunctions)
	result := unreachableDeclarations(nodes, roots)
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fail(err)
	}
}

func readDeadFunctions() map[string]bool {
	var deadPackages []deadcodePackage
	if err := json.NewDecoder(os.Stdin).Decode(&deadPackages); err != nil {
		fail(err)
	}
	deadFunctions := make(map[string]bool)
	for _, pkg := range deadPackages {
		for _, function := range pkg.Funcs {
			deadFunctions[reportedFunctionKey(function.Position, function.Name)] = true
		}
	}
	return deadFunctions
}

func loadPackages(patterns []string) (string, []*packages.Package) {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	loaded, err := packages.Load(&packages.Config{
		Dir: root,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedImports | packages.NeedDeps,
		Tests: true,
	}, patterns...)
	if err != nil {
		fail(err)
	}
	if packages.PrintErrors(loaded) > 0 {
		os.Exit(1)
	}
	return root, loaded
}

func buildGraph(
	root string,
	loaded []*packages.Package,
	deadFunctions map[string]bool,
) (map[string]*node, map[string]bool) {
	nodes := make(map[string]*node)
	roots := make(map[string]bool)
	for _, pkg := range loaded {
		if pkg.Types == nil || pkg.TypesInfo == nil || !repositoryPackage(pkg.Types.Path()) {
			continue
		}
		for _, file := range pkg.Syntax {
			path := relativePath(root, pkg.Fset.Position(file.Pos()).Filename)
			generated := ast.IsGenerated(file)
			for _, item := range file.Decls {
				switch value := item.(type) {
				case *ast.FuncDecl:
					if !deadFunctions[deadFunctionKey(pkg, value, path)] {
						addUses(roots, pkg, value)
					}
				case *ast.GenDecl:
					addGeneralDeclaration(nodes, roots, pkg, value, path, generated)
				}
			}
		}
	}
	return nodes, roots
}

func unreachableDeclarations(
	nodes map[string]*node,
	roots map[string]bool,
) []declaration {
	reachable := make(map[string]bool)
	queue := make([]string, 0, len(roots))
	for key := range roots {
		queue = append(queue, key)
	}
	for len(queue) > 0 {
		key := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if reachable[key] {
			continue
		}
		reachable[key] = true
		if current := nodes[key]; current != nil {
			for edge := range current.edges {
				queue = append(queue, edge)
			}
		}
	}

	result := make([]declaration, 0)
	for key, current := range nodes {
		if !reachable[key] {
			result = append(result, current.declaration)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if left.Position.File != right.Position.File {
			return left.Position.File < right.Position.File
		}
		if left.Position.Line != right.Position.Line {
			return left.Position.Line < right.Position.Line
		}
		return left.Name < right.Name
	})
	return result
}

func addGeneralDeclaration(
	nodes map[string]*node,
	roots map[string]bool,
	pkg *packages.Package,
	declarationValue *ast.GenDecl,
	path string,
	generated bool,
) {
	previousConstUses := make(map[string]bool)
	for _, specification := range declarationValue.Specs {
		switch value := specification.(type) {
		case *ast.TypeSpec:
			addObject(nodes, pkg, pkg.TypesInfo.Defs[value.Name], value, path, generated)
		case *ast.ValueSpec:
			previousConstUses = addValueSpecification(
				nodes, roots, pkg, declarationValue.Tok, value,
				path, generated, previousConstUses,
			)
		}
	}
}

func addValueSpecification(
	nodes map[string]*node,
	roots map[string]bool,
	pkg *packages.Package,
	tokenValue token.Token,
	value *ast.ValueSpec,
	path string,
	generated bool,
	previousConstUses map[string]bool,
) map[string]bool {
	implicitConst := tokenValue == token.CONST &&
		value.Type == nil && len(value.Values) == 0
	for _, name := range value.Names {
		object := pkg.TypesInfo.Defs[name]
		addObject(nodes, pkg, object, value, path, generated)
		if implicitConst {
			addEdges(nodes, object, previousConstUses)
		}
	}
	if tokenValue == token.CONST && !implicitConst {
		currentConstUses := make(map[string]bool)
		addUses(currentConstUses, pkg, value)
		return currentConstUses
	}
	if tokenValue == token.VAR {
		for _, expression := range value.Values {
			addUses(roots, pkg, expression)
		}
	}
	return previousConstUses
}

func addEdges(nodes map[string]*node, object types.Object, edges map[string]bool) {
	key, _ := objectKey(object)
	current := nodes[key]
	if current == nil {
		return
	}
	for dependency := range edges {
		current.edges[dependency] = true
	}
}

func addObject(
	nodes map[string]*node,
	pkg *packages.Package,
	object types.Object,
	syntax ast.Node,
	path string,
	generated bool,
) {
	key, kind := objectKey(object)
	if key == "" || object.Name() == "_" {
		return
	}
	current := nodes[key]
	if current == nil {
		positionValue := pkg.Fset.Position(object.Pos())
		current = &node{
			declaration: declaration{
				Name: object.Name(), Kind: kind,
				Position: position{
					File: path, Line: positionValue.Line, Col: positionValue.Column,
				},
				Generated: generated,
			},
			edges: make(map[string]bool),
		}
		nodes[key] = current
	}
	addUses(current.edges, pkg, syntax)
}

func addUses(target map[string]bool, pkg *packages.Package, syntax ast.Node) {
	ast.Inspect(syntax, func(item ast.Node) bool {
		identifier, ok := item.(*ast.Ident)
		if !ok {
			return true
		}
		key, _ := objectKey(pkg.TypesInfo.Uses[identifier])
		if key != "" {
			target[key] = true
		}
		return true
	})
}

func objectKey(object types.Object) (string, string) {
	if object == nil || object.Pkg() == nil || object.Parent() != object.Pkg().Scope() ||
		!repositoryPackage(object.Pkg().Path()) {
		return "", ""
	}
	switch object.(type) {
	case *types.TypeName:
		return object.Pkg().Path() + ":" + object.Name(), "type"
	case *types.Var:
		return object.Pkg().Path() + ":" + object.Name(), "var"
	case *types.Const:
		return object.Pkg().Path() + ":" + object.Name(), "const"
	default:
		return "", ""
	}
}

func repositoryPackage(path string) bool {
	return path == "tgo" || strings.HasPrefix(path, "tgo/")
}

func deadFunctionKey(pkg *packages.Package, declarationValue *ast.FuncDecl, path string) string {
	positionValue := pkg.Fset.Position(declarationValue.Name.Pos())
	return deadFunctionKeyValue(
		path,
		positionValue.Line,
		positionValue.Column,
		functionName(pkg, declarationValue),
	)
}

func functionName(pkg *packages.Package, declarationValue *ast.FuncDecl) string {
	name := declarationValue.Name.Name
	object, ok := pkg.TypesInfo.Defs[declarationValue.Name].(*types.Func)
	if !ok {
		return name
	}
	signature, ok := object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return name
	}
	typ := signature.Recv().Type()
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	named, ok := typ.(*types.Named)
	if !ok {
		return name
	}
	return named.Obj().Name() + "." + name
}

func deadFunctionKeyValue(file string, line int, column int, name string) string {
	return fmt.Sprintf("%s:%d:%d:%s", file, line, column, name)
}

func reportedFunctionKey(positionValue position, name string) string {
	return deadFunctionKeyValue(
		positionValue.File,
		positionValue.Line,
		positionValue.Col,
		name,
	)
}

func relativePath(root string, path string) string {
	result, err := filepath.Rel(root, path)
	if err != nil {
		fail(err)
	}
	return filepath.ToSlash(result)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
