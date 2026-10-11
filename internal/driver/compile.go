package driver

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lucasavila00/tgo/internal/checker"
	"github.com/lucasavila00/tgo/internal/lowering"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// Compile translates the module's TGo files and returns file contents by module-relative path.
// It keeps other module files in the result and panics on invalid source or compiler failures.
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
	tgoFiles := make(map[string]string)
	for _, name := range names {
		content := files[name]
		if strings.HasSuffix(name, ".tgo") {
			goName := generatedName(name)
			tgoFiles[goName] = name
			name = goName
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
	selected := loadMetadata(work)
	loader := checker.NewLoader(work, selected, tgoFiles, used)
	activeFiles := make(map[string]bool)
	var checked []*checker.Package
	for _, metadata := range selected {
		if metadata.Module == nil || !metadata.Module.Main {
			continue
		}
		pkg, err := loader.Check(metadata)
		must(err)
		checked = append(checked, pkg)
		for _, filename := range pkg.GoFiles {
			name, err := filepath.Rel(work, filename)
			must(err)
			if tgoFiles[name] != "" {
				activeFiles[name] = true
			}
		}
	}
	for _, pkg := range checked {
		if len(pkg.Operations) == 0 {
			continue
		}
		rejectUnsafePropagation(pkg)
		operations := make(map[string]lowering.Propagation)
		for name, operation := range pkg.Operations {
			if !operation.Conditional {
				operations[name] = lowering.Propagation{Wrap: operation.Wrap, CallName: operation.CallName, Position: operation.Position}
			}
		}
		source, err := lowering.SourceFromCheckedPackage(pkg, work)
		must(err)
		output, err := source.LowerPropagations(operations)
		must(err)
		for name, content := range output {
			must(os.WriteFile(name, content, 0600))
		}
	}
	// Check the emitted Go with the standard checker after all source packages
	// have been checked and lowered.
	for _, pkg := range loadProject(work) {
		if len(pkg.Errors) != 0 {
			panic(fmt.Sprintf("TGo: generated Go: %s", pkg.Errors[0]))
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

func loadMetadata(dir string) []*packages.Package {
	roots, err := packages.Load(&packages.Config{Dir: dir, Tests: true, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedExportFile | packages.NeedModule}, "./...")
	must(err)
	var result []*packages.Package
	packages.Visit(roots, nil, func(pkg *packages.Package) { result = append(result, pkg) })
	return result
}
