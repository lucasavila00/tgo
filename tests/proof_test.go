package tests

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

	"github.com/lucasavila00/tgo/internal/lowering"
)

var updateManifest = flag.Bool("update-manifest", false, "update expression locations after a fixture change")

type proofMode uint8

const (
	proofCompile proofMode = iota
	proofNeutral
	proofReturn
)

// TestProofCompile checks that statement and return insertions compile at every eligible expression site.
func TestProofCompile(t *testing.T) {
	proofMatrix(t, proofCompile)
}

// TestProofNeutral checks that trace-marker insertions preserve runtime behavior and execute at the expected sites.
func TestProofNeutral(t *testing.T) {
	proofMatrix(t, proofNeutral)
}

// TestProofReturn checks early-return traces, panic timing, and error identity at eligible expression sites.
func TestProofReturn(t *testing.T) {
	proofMatrix(t, proofReturn)
}

// proofMatrix builds and tests insertion variants for each eligible expression in the proof fixtures.
func proofMatrix(t *testing.T, mode proofMode) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	rejections := readProofRejections(t)
	dir := proofDirectory(t)
	module := fmt.Sprintf("module example.com/variants\n\ngo 1.27.1\n\nrequire "+
		"github.com/lucasavila00/tgo v0.0.0\nreplace github.com/lucasavila00/tgo => "+
		"%s\n",
		filepath.Join(root, ".."))
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, pattern := range []string{"./testdata/proof", "./testdata/proof/foreign"} {
		source, err := lowering.Load(root, pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, site := range source.Sites {
			if site.Reason != "" {
				continue
			}
			t.Run(fmt.Sprintf("%s:%d", filepath.Base(site.File), site.Start), func(t *testing.T) {
				count = checkProofSite(t, source, site, dir, count, rejections, mode)
			})
		}
	}
	if count == 0 {
		t.Skip("no expression sites selected")
	}
	checkProofVariants(t, dir)
}

// readProofRejections reads the expected insertion diagnostics keyed by expression location.
func readProofRejections(t *testing.T) map[string]string {
	t.Helper()
	rejectionsData, err := os.ReadFile("testdata/proof/rejections.json")
	if err != nil {
		t.Fatal(err)
	}
	var rejections map[string]string
	if err := json.Unmarshal(rejectionsData, &rejections); err != nil {
		t.Fatal(err)
	}
	return rejections
}

// writeProofVariant writes one transformed package and its runtime checks, and returns the variant directory.
func writeProofVariant(t *testing.T,
	dir string,
	count int,
	files map[string][]byte,
	packageName string,
	site lowering.Site,
	mode proofMode) string {
	t.Helper()
	variant := filepath.Join(dir, fmt.Sprintf("v%d", count))
	if err := os.Mkdir(variant, 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(variant, filepath.Base(name)), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if mode == proofNeutral {
		writeNeutralChecks(t, variant, packageName, site)
	}
	if mode == proofReturn {
		writeReturnChecks(t, variant, site)
	}
	return variant
}

// proofDirectory creates a work directory, using the requested artifact folder when set.
func proofDirectory(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if artifacts := os.Getenv("TGO_PROOF_ARTIFACTS"); artifacts != "" {
		dir = filepath.Join(artifacts, t.Name())
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// proofStatements builds the statements to insert for compile, trace-marker, or early-return checks.
func proofStatements(mode proofMode, returning bool) func([]ast.Expr) []ast.Stmt {
	if returning {
		return func([]ast.Expr) []ast.Stmt {
			exit := &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}}
			if mode != proofReturn {
				return []ast.Stmt{exit}
			}
			return []ast.Stmt{
				&ast.IncDecStmt{X: ast.NewIdent("proofVisits"), Tok: token.INC},
				&ast.IfStmt{Cond: &ast.BinaryExpr{X: ast.NewIdent("proofVisits"), Op: token.EQL,
					Y: ast.NewIdent("proofExit")}, Body: &ast.BlockStmt{List: []ast.Stmt{exit}}},
			}
		}
	}
	if mode == proofNeutral {
		return func([]ast.Expr) []ast.Stmt {
			return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("proofMark")}}}
		}
	}
	return func([]ast.Expr) []ast.Stmt {
		return []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0"}}}}
	}
}

// checkProofVariants runs Go tests for all generated variants and saves results when artifacts are requested.
func checkProofVariants(t *testing.T, dir string) {
	t.Helper()
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

// writeNeutralChecks writes tests that compare execution traces and marker positions with the reference.
func writeNeutralChecks(t *testing.T, dir, packageName string, site lowering.Site) {
	t.Helper()
	if packageName == "foreign" {
		checks := `package foreign
import "testing"
var proofMarks int
func proofMark() {proofMarks++}
func TestNeutral(t *testing.T) {v:=Value(1);if int(v)!=1 {t.Fatal(v)};` +
			`Use(v);if proofMarks!=1 {t.Fatalf("markers: %d, want 1",proofMarks)}}
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
	checks.WriteString("package proof\nimport (\"testing\";\"reflect\";\"errors\";\"fmt\")\nvar proofMarks " +
		"[]int\nfunc proofMark(){proofMarks=append(proofMarks,len(Trace))}\nfunc " +
		"TestNeutral(t *testing.T) {sentinel:=errors.New(\"sentinel\")\n")
	for _, input := range inputs {
		fmt.Fprintf(&checks,
			"t.Run(%q,func(t *testing.T){Trace=nil;proofMarks=nil;Reset(%t);var result "+
				"error;var panicValue any;panicked:=false;func(){defer "+
				"func(){panicValue=recover();panicked=panicValue!=nil}();"+
				"result=Case%s(sentinel)}();if panicked!=%t {t.Fatalf(\"panic: %%v\","+
				"panicked)};if fmt.Sprint(panicValue)!=ReferencePanic(%q,%t) "+
				"{t.Fatalf(\"panic value: %%v\",panicValue)};if result!=nil "+
				"{t.Fatalf(\"result: %%v\",result)};if !reflect.DeepEqual(Trace,Reference(%q,"+
				"%t)){t.Fatalf(\"trace: %%v, want %%v\",Trace,Reference(%q,%t))};"+
				"t.Logf(\"marker positions: %%v\",proofMarks)})\n",
			fmt.Sprintf("%s/%t", input.Name, input.Reached), input.Reached, input.Name, input.Panic, input.Name,
			input.Panic, input.Name, input.Reached, input.Name, input.Reached)
	}
	fmt.Fprintf(&checks, "for _,input:=range []struct{name string;reached bool}{")
	for _, input := range inputs {
		fmt.Fprintf(&checks, "{%q,%t},", input.Name, input.Reached)
	}
	checks.WriteString("}{")
	checks.WriteString("Trace=nil;proofMarks=nil;Reset(input.reached);func(){defer " +
		"func(){recover()}();switch input.name{")
	seen := map[string]bool{}
	for _, input := range inputs {
		if !seen[input.Name] {
			fmt.Fprintf(&checks, "case %q:Case%s(sentinel);", input.Name, input.Name)
			seen[input.Name] = true
		}
	}
	checks.WriteString("}}();")
	fmt.Fprintf(&checks,
		"expected,ok:=ReferenceMarkers(%q,%d,%d,input.name,input.reached);if !ok "+
			"{t.Fatal(\"missing marker oracle\")};if !reflect.DeepEqual(proofMarks,"+
			"expected){t.Fatalf(\"%%s marker positions: %%v, want %%v\",input.name,"+
			"proofMarks,expected)}}\n",
		filepath.Base(site.File), site.Start, site.End)
	checks.WriteString("}\n")
	if err := os.WriteFile(filepath.Join(dir, "proof_test.go"), []byte(checks.String()), 0600); err != nil {
		t.Fatal(err)
	}
}

// TestProofInventory checks the recorded expression locations and updates the manifest when requested.
func TestProofInventory(t *testing.T) {
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	var sites []lowering.Site
	for _, pattern := range []string{"./testdata/proof", "./testdata/proof/foreign"} {
		source, err := lowering.Load(dir, pattern)
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

// TestProofReference checks that the original proof fixtures match their independent execution traces.
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
	module := fmt.Sprintf("module example.com/proof\n\ngo 1.27.1\n\nrequire github.com/lucasavila00/tgo "+
		"v0.0.0\nreplace github.com/lucasavila00/tgo => %s\n",
		filepath.Join(root, ".."))
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
	checks.WriteString("package proof\nimport (\"testing\";\"reflect\";\"errors\")\nfunc TestReference(t " +
		"*testing.T) { sentinel:=errors.New(\"sentinel\")\n")
	for _, input := range inputs {
		fmt.Fprintf(&checks,
			"t.Run(%q,func(t *testing.T) { Reset(%t); panicked:=false; var result error;"+
				" func(){defer func(){if recover()!=nil {panicked=true}}();"+
				"result=Case%s(sentinel)}(); if panicked!=%t {t.Fatalf(\"panic: %%v\","+
				"panicked)};if result!=nil {t.Fatalf(\"unexpected result: %%v\",result)}; if "+
				"!reflect.DeepEqual(Trace,Reference(%q,%t)) {t.Fatalf(\"trace: %%v, want "+
				"%%v\",Trace,Reference(%q,%t))} })\n",
			fmt.Sprintf("%s/%t", input.Name, input.Reached), input.Reached, input.Name, input.Panic, input.Name,
			input.Reached, input.Name, input.Reached)
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

// checkProofSite checks an expression against expected rejections and writes its accepted insertion variants.
// It returns the updated variant count.
func checkProofSite(
	t *testing.T,
	source *lowering.Source,
	site lowering.Site,
	dir string,
	count int,
	rejections map[string]string,
	mode proofMode,
) int {
	t.Helper()
	for _, returning := range []bool{false, true} {
		if (mode == proofNeutral && returning) || (mode == proofReturn && !returning) {
			continue
		}
		if returning && !site.ErrorReturn {
			continue
		}
		after := proofStatements(mode, returning)
		files, err := source.InsertAfter(site, after)
		expected := rejections[fmt.Sprintf("%s:%d:%d", site.File, site.Start, site.End)]
		if expected != "" {
			if err == nil || !strings.Contains(err.Error(), expected) {
				t.Fatalf("rejection: %v, want %s", err, expected)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
		variant := writeProofVariant(t, dir, count, files, source.Package.Name, site, mode)
		t.Logf("%s: return=%t", variant, returning)
	}
	return count
}
