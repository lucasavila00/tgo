package compiler

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSourceLowersNestedMatchLabels(t *testing.T) {
	data := []byte(`package sample
type E enum { A struct{} }
func use(value E) {
Outer: Inner: match value { case A(item): break Outer }
}
`)
	source, err := parseSource(token.NewFileSet(), "sample.tgo", data)
	if err != nil {
		t.Fatal(err)
	}
	labels := 0
	var outer *ast.LabeledStmt
	ast.Inspect(source.File, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.LabeledStmt:
			labels++
			if node.Label.Name == "Outer" {
				outer = node
			}
		case *ast.EmptyStmt:
			if !node.Implicit {
				t.Fatalf("artificial statement at %v", node.Pos())
			}
		}
		return true
	})
	if outer == nil {
		t.Fatalf("lowered labels=%d outer is missing", labels)
	}
	inner, ok := outer.Stmt.(*ast.LabeledStmt)
	if !ok || inner.Label.Name != "Inner" {
		t.Fatalf("lowered labels=%d outer=%#v", labels, outer)
	}
	if _, ok := inner.Stmt.(*ast.SwitchStmt); !ok {
		t.Fatalf("inner label statement: %T", inner.Stmt)
	}
}

func TestValidationRuntimeRejectsForeignFile(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "internal", "tgoruntime")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "runtime.go")
	if err := os.WriteFile(path, []byte("package tgoruntime\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	builder := &packageBuilder{root: root}
	if err := builder.ensureValidationRuntime(); err == nil {
		t.Fatal("arbitrary runtime.go was accepted")
	}
}

func TestValidationRuntimeAcceptsCanonicalSource(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "internal", "tgoruntime")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	canonical := validationRuntime[len(generatedHeader)+1:]
	path := filepath.Join(directory, "runtime.go")
	if err := os.WriteFile(path, canonical, 0o644); err != nil {
		t.Fatal(err)
	}
	builder := &packageBuilder{root: root}
	if err := builder.ensureValidationRuntime(); err != nil {
		t.Fatalf("canonical runtime.go: %v", err)
	}
}

func TestValidationRuntimeRejectsExtraFileBesideCanonicalSource(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "internal", "tgoruntime")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	canonical := validationRuntime[len(generatedHeader)+1:]
	if err := os.WriteFile(
		filepath.Join(directory, "runtime.go"), canonical, 0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "extra.go"), []byte("package tgoruntime\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	builder := &packageBuilder{root: root}
	if err := builder.ensureValidationRuntime(); err == nil {
		t.Fatal("extra runtime Go file was accepted")
	}
}
