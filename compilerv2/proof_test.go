package compilerv2

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var updateManifest = flag.Bool("update-manifest", false, "update expression locations after a fixture change")

func TestProofCompile(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/variants\n\ngo 1.27.1\n\nrequire github.com/lucasavila00/tgo/compilerv2 v0.0.0\nreplace github.com/lucasavila00/tgo/compilerv2 => %s\n", root)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, pattern := range []string{"./testdata/proof", "./testdata/proof/foreign"} {
		source, err := Load(root, pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, site := range source.Sites {
			if site.Reason != "" {
				continue
			}
			t.Run(fmt.Sprintf("%s:%d", filepath.Base(site.File), site.Start), func(t *testing.T) {
				for _, returning := range []bool{false, true} {
					if returning && !site.ErrorReturn {
						continue
					}
					after := func([]ast.Expr) []ast.Stmt {
						return []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{integer(0)}}}
					}
					if returning {
						after = func([]ast.Expr) []ast.Stmt {
							return []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}}}
						}
					}
					files, err := source.InsertAfter(site, after)
					if err != nil {
						t.Fatal(err)
					}
					count++
					variant := filepath.Join(dir, fmt.Sprintf("v%d", count))
					if err := os.Mkdir(variant, 0700); err != nil {
						t.Fatal(err)
					}
					for name, content := range files {
						if err := os.WriteFile(filepath.Join(variant, filepath.Base(name)), content, 0600); err != nil {
							t.Fatal(err)
						}
					}
					t.Logf("%s: return=%t", variant, returning)
				}
			})
		}
	}
	if count == 0 {
		t.Skip("no expression sites selected")
	}
	cmd := exec.Command("go", "test", "-p", "4", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compile expression variants: %v\n%s", err, output)
	}
}

func TestProofInventory(t *testing.T) {
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	var sites []Site
	for _, pattern := range []string{"./testdata/proof", "./testdata/proof/foreign"} {
		source, err := Load(dir, pattern)
		if err != nil {
			t.Fatal(err)
		}
		sites = append(sites, source.Sites...)
	}
	actual, err := json.MarshalIndent(sites, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	actual = append(actual, '\n')
	path := "testdata/proof/manifest.json"
	if *updateManifest {
		if err := os.WriteFile(path, actual, 0600); err != nil {
			t.Fatal(err)
		}
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("expression inventory changed; inspect the fixtures and update manifest.json")
	}
}

type proofInput struct {
	Name    string `json:"name"`
	Reached bool   `json:"reached"`
	Panic   bool   `json:"panic"`
}

func TestProofReference(t *testing.T) {
	data, err := os.ReadFile("testdata/proof/expect.json")
	if err != nil {
		t.Fatal(err)
	}
	var inputs []proofInput
	if err := json.Unmarshal(data, &inputs); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	module := fmt.Sprintf("module example.com/proof\n\ngo 1.27.1\n\nrequire github.com/lucasavila00/tgo/compilerv2 v0.0.0\nreplace github.com/lucasavila00/tgo/compilerv2 => %s\n", root)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("testdata/proof/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(file) == "reference.go" {
			content = []byte(strings.TrimPrefix(string(content), "//go:build proofreference\n"))
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(file)), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var checks strings.Builder
	checks.WriteString("package proof\nimport (\"testing\";\"reflect\";\"errors\")\nfunc TestReference(t *testing.T) { sentinel:=errors.New(\"sentinel\")\n")
	for _, input := range inputs {
		fmt.Fprintf(&checks, "t.Run(%q,func(t *testing.T) { Reset(%t); panicked:=false; var result error; func(){defer func(){if recover()!=nil {panicked=true}}();result=Case%s(sentinel)}(); if panicked!=%t {t.Fatalf(\"panic: %%v\",panicked)};if result!=nil {t.Fatalf(\"unexpected result: %%v\",result)}; if !reflect.DeepEqual(Trace,Reference(%q,%t)) {t.Fatalf(\"trace: %%v, want %%v\",Trace,Reference(%q,%t))} })\n", fmt.Sprintf("%s/%t", input.Name, input.Reached), input.Reached, input.Name, input.Panic, input.Name, input.Reached, input.Name, input.Reached)
	}
	checks.WriteString("}\n")
	if err := os.WriteFile(filepath.Join(dir, "proof_test.go"), []byte(checks.String()), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-run", "^TestReference$", "-count=1")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("proof references: %v\n%s", err, output)
	}
}
