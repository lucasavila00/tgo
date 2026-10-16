package lowering

import (
	"github.com/lucasavila00/tgo/internal/ast"
	stockast "go/ast"
	stocktypes "go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// TestLoadKeepsStockTypeObjects checks stock Go types stay separate from private lowering types.
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
	fn := source.Package.Syntax[0].Decls[0].(*stockast.FuncDecl)
	errorType := fn.Type.Results.List[0].Type
	if source.Package.TypesInfo.TypeOf(errorType) != stocktypes.Universe.Lookup("error").Type() {
		t.Fatal("public Load changed stock Go type identity")
	}
	privateFunction := source.inventory.Checked.Syntax[0].Decls[0].(*ast.FuncDecl)
	privateErrorType := privateFunction.Type.Results.List[0].Type
	privateOffset := source.inventory.Checked.Fset.Position(privateErrorType.Pos()).Offset
	stockOffset := source.Package.Fset.Position(errorType.Pos()).Offset
	if privateOffset != stockOffset {
		t.Fatal("private checking changed source offsets")
	}
	if source.inventory.Checked.TypesInfo.TypeOf(privateErrorType) != types.Universe.Lookup("error").Type() {
		t.Fatal("lowering did not use private source type objects")
	}
	if source.Package.TypesInfo.Defs[fn.Name].Pkg() != source.Package.Types {
		t.Fatal("public bindings changed package identity")
	}
	if source.inventory.Checked.TypesInfo.Defs[privateFunction.Name].Pkg() != source.inventory.Checked.Types {
		t.Fatal("private bindings changed package identity")
	}
}
