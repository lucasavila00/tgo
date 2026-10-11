package lowering

import (
	"go/ast"
	stocktypes "go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func TestLoadKeepsStockTypeObjects(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":  "module example.com/identity\n\ngo 1.27.1\n",
		"main.go": "package example\nfunc Value() error { return nil }\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, err := Load(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	if source.Package.Fset != source.checked.Fset || source.Package.Syntax[0] != source.checked.Syntax[0] {
		t.Fatal("checks must use the same source syntax and positions")
	}
	fn := source.Package.Syntax[0].Decls[0].(*ast.FuncDecl)
	errorType := fn.Type.Results.List[0].Type
	if source.Package.TypesInfo.TypeOf(errorType) != stocktypes.Universe.Lookup("error").Type() {
		t.Fatal("public Load changed stock Go type identity")
	}
	if source.checked.TypesInfo.TypeOf(errorType) != types.Universe.Lookup("error").Type() {
		t.Fatal("lowering did not use private source type objects")
	}
	if source.Package.TypesInfo.Defs[fn.Name].Pkg() != source.Package.Types {
		t.Fatal("public bindings changed package identity")
	}
	if source.checked.TypesInfo.Defs[fn.Name].Pkg() != source.checked.Types {
		t.Fatal("private bindings changed package identity")
	}
}
