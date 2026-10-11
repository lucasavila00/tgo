package driver

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// moduleFiles contains owned Go and TGo source, keyed by module-relative path.
type moduleFiles struct {
	root  string
	files map[string][]byte
	names []string
}

// preparedModule records generated paths staged in the source module.
type preparedModule struct {
	dir      string
	tgoFiles map[string]string
	used     map[string]bool
}

// findModule returns the module root that Go selects for dir.
func findModule(dir string) string {
	dir, err := filepath.Abs(dir)
	must(err)
	command := exec.Command("go", "env", "GOMOD")
	command.Dir = dir
	module, err := command.Output()
	must(err)
	moduleFile := strings.TrimSpace(string(module))
	if moduleFile == "" || moduleFile == os.DevNull {
		panic("TGo requires a Go module")
	}
	return filepath.Dir(moduleFile)
}

// readModule reads owned source without stale generated files or asset contents.
func readModule(root string) *moduleFiles {
	paths := make(map[string]bool)
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
		paths[name] = true
		return nil
	}))
	// Source files control compilation; matching generated files can be stale.
	for name := range paths {
		if strings.HasSuffix(name, ".tgo") && ownsModuleFile(paths, name) {
			delete(paths, generatedName(name))
		}
	}
	files := make(map[string][]byte)
	var names []string
	for name := range paths {
		if !ownsModuleFile(paths, name) || !(strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".tgo")) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, name))
		must(err)
		files[name] = content
		names = append(names, name)
	}
	slices.Sort(names)
	return &moduleFiles{root: root, files: files, names: names}
}

// prepareModule writes original TGo at generated Go paths for package selection.
func prepareModule(source *moduleFiles) *preparedModule {
	used := make(map[string]bool)
	// Reserve source identifiers before the compiler adds names.
	for _, content := range source.files {
		for _, word := range strings.FieldsFunc(string(content), func(r rune) bool {
			return !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
		}) {
			used[word] = true
		}
	}
	tgoFiles := make(map[string]string)
	for _, name := range source.names {
		if !strings.HasSuffix(name, ".tgo") {
			continue
		}
		goName := generatedName(name)
		tgoFiles[goName] = name
		must(os.WriteFile(filepath.Join(source.root, goName), source.files[name], 0644))
	}
	return &preparedModule{dir: source.root, tgoFiles: tgoFiles, used: used}
}

// ownsModuleFile excludes data directories, dependencies, and nested modules.
func ownsModuleFile(files map[string]bool, name string) bool {
	for dir := filepath.Dir(name); dir != "."; dir = filepath.Dir(dir) {
		base := filepath.Base(dir)
		if base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") {
			return false
		}
		if files[filepath.Join(dir, "go.mod")] {
			return false
		}
	}
	return true
}
