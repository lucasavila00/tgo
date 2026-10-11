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
	proofMatrix(t, false, false)
}

func TestProofNeutral(t *testing.T) {
	proofMatrix(t, true, false)
}

func TestProofReturn(t *testing.T) {
	proofMatrix(t, false, true)
}

func proofMatrix(t *testing.T, neutral, returnCheck bool) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if artifacts := os.Getenv("TGO_PROOF_ARTIFACTS"); artifacts != "" {
		dir = filepath.Join(artifacts, t.Name())
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
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
					if (neutral && returning) || (returnCheck && !returning) {
						continue
					}
					if returning && !site.ErrorReturn {
						continue
					}
					after := func([]ast.Expr) []ast.Stmt {
						return []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{integer(0)}}}
					}
					if neutral {
						after = func([]ast.Expr) []ast.Stmt {
							return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("proofMark")}}}
						}
					}
					if returning {
						after = func([]ast.Expr) []ast.Stmt {
							exit := &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}}
							if returnCheck {
								return []ast.Stmt{
									&ast.IncDecStmt{X: ast.NewIdent("proofVisits"), Tok: token.INC},
									&ast.IfStmt{Cond: &ast.BinaryExpr{X: ast.NewIdent("proofVisits"), Op: token.EQL, Y: ast.NewIdent("proofExit")}, Body: &ast.BlockStmt{List: []ast.Stmt{exit}}},
								}
							}
							return []ast.Stmt{exit}
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
					if neutral {
						writeNeutralChecks(t, variant, source.Package.Name, site)
					}
					if returnCheck {
						writeReturnChecks(t, variant, site)
					}
					t.Logf("%s: return=%t", variant, returning)
				}
			})
		}
	}
	if count == 0 {
		t.Skip("no expression sites selected")
	}
	cmd := exec.Command("go", "test", "-p", "4", "-json", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if artifacts := os.Getenv("TGO_PROOF_ARTIFACTS"); artifacts != "" {
		if writeErr := os.WriteFile(filepath.Join(dir, "results.json"), output, 0600); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err != nil {
		t.Fatalf("compile expression variants: %v\n%s", err, output)
	}
}

func writeNeutralChecks(t *testing.T, dir, packageName string, site Site) {
	t.Helper()
	if packageName == "foreign" {
		checks := `package foreign
import "testing"
var proofMarks int
func proofMark() {proofMarks++}
func TestNeutral(t *testing.T) {v:=Value(1);if int(v)!=1 {t.Fatal(v)};Use(v);if proofMarks!=1 {t.Fatalf("markers: %d, want 1",proofMarks)}}
`
		if err := os.WriteFile(filepath.Join(dir, "proof_test.go"), []byte(checks), 0600); err != nil {
			t.Fatal(err)
		}
		return
	}
	reference, err := os.ReadFile("testdata/proof/reference.go")
	if err != nil {
		t.Fatal(err)
	}
	reference = []byte(strings.TrimPrefix(string(reference), "//go:build proofreference\n"))
	if err := os.WriteFile(filepath.Join(dir, "reference.go"), reference, 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/proof/expect.json")
	if err != nil {
		t.Fatal(err)
	}
	var inputs []proofInput
	if err := json.Unmarshal(data, &inputs); err != nil {
		t.Fatal(err)
	}
	var checks strings.Builder
	checks.WriteString("package proof\nimport (\"testing\";\"reflect\";\"errors\")\nvar proofMarks []int\nfunc proofMark(){proofMarks=append(proofMarks,len(Trace))}\nfunc TestNeutral(t *testing.T) {sentinel:=errors.New(\"sentinel\")\n")
	for _, input := range inputs {
		fmt.Fprintf(&checks, "t.Run(%q,func(t *testing.T){Trace=nil;proofMarks=nil;Reset(%t);var result error;panicked:=false;func(){defer func(){if recover()!=nil {panicked=true}}();result=Case%s(sentinel)}();if panicked!=%t {t.Fatalf(\"panic: %%v\",panicked)};if result!=nil {t.Fatalf(\"result: %%v\",result)};if !reflect.DeepEqual(Trace,Reference(%q,%t)){t.Fatalf(\"trace: %%v, want %%v\",Trace,Reference(%q,%t))};t.Logf(\"marker positions: %%v\",proofMarks)})\n", fmt.Sprintf("%s/%t", input.Name, input.Reached), input.Reached, input.Name, input.Panic, input.Name, input.Reached, input.Name, input.Reached)
	}
	fmt.Fprintf(&checks, "for _,input:=range []struct{name string;reached bool}{")
	for _, input := range inputs {
		fmt.Fprintf(&checks, "{%q,%t},", input.Name, input.Reached)
	}
	checks.WriteString("}{")
	checks.WriteString("Trace=nil;proofMarks=nil;Reset(input.reached);func(){defer func(){recover()}();switch input.name{")
	seen := map[string]bool{}
	for _, input := range inputs {
		if !seen[input.Name] {
			fmt.Fprintf(&checks, "case %q:Case%s(sentinel);", input.Name, input.Name)
			seen[input.Name] = true
		}
	}
	checks.WriteString("}}();")
	fmt.Fprintf(&checks, "if expected,ok:=ReferenceMarkers(%q,%d,%d,input.name,input.reached);!ok {t.Fatal(\"missing marker oracle\")};if !reflect.DeepEqual(proofMarks,expected){t.Fatalf(\"%%s marker positions: %%v, want %%v\",input.name,proofMarks,expected)}}\n", filepath.Base(site.File), site.Start, site.End)
	checks.WriteString("}\n")
	if err := os.WriteFile(filepath.Join(dir, "proof_test.go"), []byte(checks.String()), 0600); err != nil {
		t.Fatal(err)
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
