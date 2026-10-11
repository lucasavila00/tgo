package driver

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
	"slices"
	"strings"

	"github.com/lucasavila00/tgo/internal/frontend"
	"github.com/lucasavila00/tgo/internal/lowering"

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
	work, err := os.MkdirTemp("", "tgo-transpiler-")
	must(err)
	defer os.RemoveAll(work)
	operations := make(map[string]frontend.Marker)
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
	// Source files control compilation; matching generated files can be stale.
	for name := range files {
		if strings.HasSuffix(name, ".tgo") {
			delete(files, generatedName(name))
		}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	// Reserve source identifiers before the compiler adds names.
	for _, content := range files {
		for _, word := range strings.FieldsFunc(string(content), func(r rune) bool {
			return !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
		}) {
			used[word] = true
		}
	}
	tgoFiles := make(map[string]bool)
	for _, name := range names {
		content := files[name]
		if strings.HasSuffix(name, ".tgo") {
			goName := generatedName(name)
			content = frontend.MarkPostfix(name, content, used, operations)
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
	initial := loadProject(work)
	activeFiles := make(map[string]bool)
	activeOperations := make(map[string]frontend.Marker)
	for _, pkg := range initial {
		for _, filename := range pkg.GoFiles {
			name, err := filepath.Rel(work, filename)
			must(err)
			if tgoFiles[name] {
				activeFiles[name] = true
			}
		}
		for _, file := range pkg.Syntax {
			name, err := filepath.Rel(work, pkg.Fset.Position(file.Pos()).Filename)
			must(err)
			if !tgoFiles[name] {
				continue
			}
			activeFiles[name] = true
			for _, operation := range frontend.MarkerExpressions(pkg.Fset, file, operations) {
				activeOperations[operation.Name] = operation
			}
		}
	}
	operations = activeOperations
	remaining := len(operations)
	for remaining > 0 {
		pkgs := loadProject(work)
		progress := false
		written := make(map[string]bool)
		for _, pkg := range pkgs {
			for _, file := range pkg.Syntax {
				filename := pkg.Fset.Position(file.Pos()).Filename
				if written[filename] {
					continue
				}
				candidates := frontend.MarkerExpressions(pkg.Fset, file, operations)
				var selected ast.Expr
				var operation frontend.Marker
				for expression, op := range candidates {
					operations[op.Name] = op
					typ := pkg.TypesInfo.TypeOf(ast.Unparen(expression))
					if typ == nil || typ == types.Typ[types.Invalid] {
						continue
					}
					if selected == nil || expression.End()-expression.Pos() < selected.End()-selected.Pos() || expression.End()-expression.Pos() == selected.End()-selected.Pos() && expression.Pos() < selected.Pos() {
						selected = expression
						operation = op
					}
				}
				if selected == nil {
					continue
				}
				result := pkg.TypesInfo.TypeOf(ast.Unparen(selected))
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
				frontend.WrapMarker(file, selected, operation)
				file.Decls = append(file.Decls, frontend.MarkerDeclaration(operation, count-1))
				writeSyntax(pkg.Fset, file)
				written[filename] = true
				operations[operation.Name] = operation
				remaining--
				progress = true
			}
		}
		if !progress {
			panic("cannot type check propagation source")
		}
	}
	loweredOperations := make(map[string]lowering.Propagation, len(operations))
	for name, operation := range operations {
		loweredOperations[name] = lowering.Propagation{Wrap: operation.Wrap, CallName: operation.CallName, Position: operation.Position}
	}
	for _, pkg := range loadProject(work) {
		changed := false
		for _, filename := range pkg.GoFiles {
			name, err := filepath.Rel(work, filename)
			must(err)
			changed = changed || activeFiles[name]
		}
		if !changed {
			continue
		}
		rejectUnsafePropagation(pkg, operations)
		checkPropagationReturns(pkg, operations)
		source, err := lowering.SourceFromPackage(pkg, work)
		must(err)
		output, err := source.LowerPropagations(loweredOperations)
		must(err)
		for name, content := range output {
			must(os.WriteFile(name, content, 0600))
		}
	}

	output := make(map[string][]byte)
	for name, content := range files {
		if strings.HasSuffix(name, ".tgo") {
			goName := generatedName(name)
			output[name] = content
			if !activeFiles[goName] {
				continue
			}
			content, err = os.ReadFile(filepath.Join(work, goName))
			must(err)
			output[goName] = content
		} else {
			if name == "go.mod" {
				content = moduleData
			}
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

func loadProject(dir string) []*packages.Package {
	roots, err := packages.Load(&packages.Config{Dir: dir, Tests: true, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps}, "./...")
	must(err)
	var result []*packages.Package
	packages.Visit(roots, nil, func(pkg *packages.Package) { result = append(result, pkg) })
	return result
}
