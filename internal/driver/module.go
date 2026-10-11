package driver

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
)

// moduleFiles contains source files, keyed by module-relative path.
type moduleFiles struct {
	root  string
	files map[string][]byte
	names []string
}

// preparedModule maps generated paths to in-memory contents for Go checking.
type preparedModule struct {
	source     *moduleFiles
	dir        string
	tgoFiles   map[string]string
	used       map[string]bool
	moduleData []byte
	overlay    map[string][]byte
	buildFlags []string
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

// readModule removes stale generated files and keeps a fixed source order.
func readModule(root string) *moduleFiles {
	files := make(map[string][]byte)
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
	return &moduleFiles{root: root, files: files, names: names}
}

// prepareModule keeps source at its original paths and prepares a Go overlay.
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
	overlay := make(map[string][]byte)
	for _, name := range source.names {
		if strings.HasSuffix(name, ".tgo") {
			goName := generatedName(name)
			tgoFiles[goName] = name
			overlay[filepath.Join(source.root, goName)] = source.files[name]
		}
	}

	// Relative replacement paths still refer to the source module's dependencies.
	moduleData, err := os.ReadFile(filepath.Join(source.root, "go.mod"))
	must(err)
	mod, err := modfile.Parse("go.mod", moduleData, nil)
	must(err)
	for _, replace := range mod.Replace {
		if replace.New.Version == "" && !filepath.IsAbs(replace.New.Path) {
			must(mod.AddReplace(replace.Old.Path, replace.Old.Version, filepath.Join(source.root, replace.New.Path), ""))
		}
	}
	moduleData, err = mod.Format()
	must(err)
	return &preparedModule{source: source, dir: source.root, overlay: overlay, tgoFiles: tgoFiles, used: used, moduleData: moduleData}
}

// collectOutput keeps source and emits Go only for selected TGo files.
func collectOutput(checked *checkedModule) map[string][]byte {
	module := checked.sources.module
	output := make(map[string][]byte)
	for name, content := range module.source.files {
		if strings.HasSuffix(name, ".tgo") {
			goName := generatedName(name)
			output[name] = content
			if !checked.activeFiles[goName] {
				continue
			}
			generated := module.overlay[filepath.Join(module.dir, goName)]
			output[goName] = generated
		} else {
			if name == "go.mod" {
				content = module.moduleData
			}
			output[name] = content
		}
	}
	return output
}
