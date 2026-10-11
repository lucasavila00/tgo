package compilerv2

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// Compile emits a Go module from the TGo files in a module directory.
// Invalid source and compiler failures panic.
func Compile(dir string) map[string][]byte {
	dir, mustErr := filepath.Abs(dir)
	must(mustErr)
	command := exec.Command("go", "env", "GOMOD")
	command.Dir = dir
	module, err := command.Output()
	must(err)
	moduleFile := strings.TrimSpace(string(module))
	if moduleFile == "" || moduleFile == os.DevNull {
		panic("TGo requires a Go module")
	}
	root := filepath.Dir(moduleFile)
	work, err := os.MkdirTemp("", "tgo-propagation-")
	must(err)
	defer os.RemoveAll(work)
	operations := make(map[string]propagation)
	files := make(map[string][]byte)
	used := make(map[string]bool)
	must(filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".worktrees" {
				return filepath.SkipDir
			}
			return nil
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[name] = content
		return nil
	}))
	// Reserve source identifiers before the compiler adds names.
	for _, content := range files {
		for _, word := range strings.FieldsFunc(string(content), func(r rune) bool {
			return !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
		}) {
			used[word] = true
		}
	}
	tgoFiles := make(map[string]bool)
	for name, content := range files {
		if strings.HasSuffix(name, ".tgo") {
			goName := strings.TrimSuffix(name, ".tgo") + ".go"
			if _, ok := files[goName]; ok {
				panic("TGo and Go files cannot share a source name")
			}
			content = markPostfix(name, content, used, operations)
			name = goName
			tgoFiles[name] = true
		}
		path := filepath.Join(work, name)
		must(os.MkdirAll(filepath.Dir(path), 0700))
		must(os.WriteFile(path, content, 0600))
	}
	// Relative replacement paths still refer to the source module's dependencies.
	moduleData, err := os.ReadFile(filepath.Join(work, "go.mod"))
	must(err)
	mod, err := modfile.Parse("go.mod", moduleData, nil)
	must(err)
	for _, replace := range mod.Replace {
		if replace.New.Version == "" && !filepath.IsAbs(replace.New.Path) {
			must(mod.AddReplace(replace.Old.Path, replace.Old.Version, filepath.Join(root, replace.New.Path), ""))
		}
	}
	moduleData, err = mod.Format()
	must(err)
	must(os.WriteFile(filepath.Join(work, "go.mod"), moduleData, 0600))
	remaining := len(operations)
	for remaining > 0 {
		pkgs, err := packages.Load(&packages.Config{Dir: work, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps}, "./...")
		must(err)
		progress := false
		for _, pkg := range pkgs {
			for _, file := range pkg.Syntax {
				candidates := markerExpressions(pkg.Fset, file, operations)
				var selected ast.Expr
				var operation propagation
				for expression, op := range candidates {
					typ := pkg.TypesInfo.TypeOf(unparen(expression))
					if typ == nil || typ == types.Typ[types.Invalid] {
						continue
					}
					if selected == nil || expression.End()-expression.Pos() < selected.End()-selected.Pos() {
						selected = expression
						operation = op
					}
				}
				if selected == nil {
					continue
				}
				result := pkg.TypesInfo.TypeOf(unparen(selected))
				count := 1
				last := result
				if tuple, ok := result.(*types.Tuple); ok {
					count = tuple.Len()
					if count == 0 {
						panic("propagation requires an error result")
					}
					last = tuple.At(count - 1).Type()
				}
				if !types.AssignableTo(last, types.Universe.Lookup("error").Type()) {
					panic("propagation requires a final error result")
				}
				wrapMarker(file, selected, operation)
				file.Decls = append(file.Decls, markerDeclaration(operation, count-1))
				writeSyntax(pkg.Fset, file)
				operations[operation.name] = operation
				remaining--
				progress = true
			}
		}
		if !progress {
			panic("cannot type check propagation source")
		}
	}
	// Type-check each changed package, then lower one call at a time.
	directories := make(map[string]bool)
	for name := range tgoFiles {
		directories[filepath.Dir(name)] = true
	}
	for directory := range directories {
		packageDir := filepath.Join(work, directory)
		for {
			source, err := Load(packageDir, ".")
			must(err)
			var selected *ast.CallExpr
			var operation propagation
			for _, file := range source.Package.Syntax {
				ast.Inspect(file, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					id, ok := call.Fun.(*ast.Ident)
					if !ok {
						return true
					}
					op, ok := operations[id.Name]
					if !ok || source.unreachable(call) {
						return true
					}
					if selected == nil || call.End()-call.Pos() < selected.End()-selected.Pos() {
						selected = call
						operation = op
					}
					return true
				})
			}
			if selected == nil {
				removeUnusedMarkers(source, operations)
				break
			}
			var site Site
			for candidate, expr := range source.expressions {
				if expr == selected {
					site = candidate
					break
				}
			}
			output, err := source.lowerPropagation(site, operation)
			must(err)
			for name, content := range output {
				must(os.WriteFile(name, content, 0600))
			}
		}
	}
	output := make(map[string][]byte)
	for name, content := range files {
		if strings.HasSuffix(name, ".tgo") {
			goName := strings.TrimSuffix(name, ".tgo") + ".go"
			content, err = os.ReadFile(filepath.Join(work, goName))
			must(err)
			output[goName] = content
		} else {
			output[name] = content
		}
	}
	return output
}

func writeSyntax(fset *token.FileSet, file *ast.File) {
	var buf bytes.Buffer
	must(format.Node(&buf, fset, file))
	must(os.WriteFile(fset.Position(file.Pos()).Filename, buf.Bytes(), 0600))
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("TGo: %v", err))
	}
}
