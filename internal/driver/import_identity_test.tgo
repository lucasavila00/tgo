package driver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceImportIdentityAcrossGoPackage(t *testing.T) {
	root := t.TempDir()
	writeImportIdentityFile(
		t, root, "go.mod", "module example.com/identity\n\ngo 1.27\n",
	)
	model := "package model\n\ntype Value struct { Text string }\n"
	writeImportIdentityFile(t, root, "model/model.tgo", model)
	writeImportIdentityFile(t, root, "model/model_tgo.go", model)
	writeImportIdentityFile(
		t,
		root,
		"bridge/bridge.go",
		"package bridge\n\n"+
			"import \"example.com/identity/model\"\n\n"+
			"func Use(value model.Value) model.Value { return value }\n",
	)
	writeImportIdentityFile(
		t,
		root,
		"app/app.tgo",
		"package app\n\n"+
			"import (\n"+
			"\t\"example.com/identity/bridge\"\n"+
			"\t\"example.com/identity/model\"\n"+
			")\n\n"+
			"func Use(value model.Value) model.Value { return bridge.Use(value) }\n",
	)
	writeImportIdentityFile(
		t,
		root,
		"app/app_test.tgo",
		"package app\n\n"+
			"import (\n"+
			"\t\"testing\"\n"+
			"\t\"example.com/identity/model\"\n"+
			")\n\n"+
			"func TestUse(t *testing.T) { _ = Use(model.Value{}) }\n",
	)
	writeImportIdentityFile(
		t,
		root,
		"app/external_test.tgo",
		"package app_test\n\n"+
			"import (\n"+
			"\t\"testing\"\n"+
			"\t\"example.com/identity/app\"\n"+
			"\t\"example.com/identity/bridge\"\n"+
			"\t\"example.com/identity/model\"\n"+
			")\n\n"+
			"func TestUse(t *testing.T) {\n"+
			"\tvalue := app.Use(model.Value{})\n"+
			"\t_ = bridge.Use(value)\n"+
			"}\n",
	)
	views, err := CompileWorkspaceViewsContext(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	production, internal, external := false, false, false
	for _, view := range views {
		if view.Package.Path == "example.com/identity/app" {
			if view.Test {
				internal = true
			} else {
				production = true
			}
		}
		if view.Package.Path == "example.com/identity/app_test" && view.External {
			external = true
		}
	}
	if !production || !internal || !external {
		t.Fatalf(
			"app views: production=%t, internal=%t, external=%t",
			production,
			internal,
			external,
		)
	}
}

func writeImportIdentityFile(t *testing.T, root, name, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
