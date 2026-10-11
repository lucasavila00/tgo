package checker

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestCheckConditionalModule(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module example.com/conditional\n\ngo 1.27.1\n",
		"main_tgo.go": `package main
import "fmt"
func choose(ok bool) string { return if ok { "yes" } else { "no" } }
func main() { fmt.Println(choose(true)) }
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	roots, err := packages.Load(&packages.Config{Dir: dir, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedExportFile | packages.NeedModule}, "./...")
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 {
		t.Fatalf("packages: %d", len(roots))
	}
	loader := NewLoader(dir, roots, map[string]string{"main_tgo.go": "main.tgo"}, make(map[string]bool))
	pkg, err := loader.Check(roots[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Conditionals) != 1 {
		t.Fatalf("conditionals: %d", len(pkg.Conditionals))
	}
}
