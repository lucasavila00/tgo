package tgolint

import (
	"bytes"
	"go/ast"
	"go/token"
	"testing"

	"tgo/pkg/syntax"

	"golang.org/x/tools/go/analysis"
)

func TestParseGeneratedMetadataUsesOnlyFixedHeaderLine(t *testing.T) {
	body := []byte("package p\n\n//tgo:v1 this is an ordinary source comment\n")
	data := []byte(generatedHeader + "\n//tgo:v2 \"model.tgo\"\n\n")
	data = append(data, body...)
	metadata := parseGeneratedMetadata(data)
	if metadata == nil || metadata.source != "model.tgo" ||
		!bytes.Equal(metadata.body, body) {
		t.Fatalf("metadata: %#v", metadata)
	}
}

func TestParseGeneratedMetadataRejectsTrailingData(t *testing.T) {
	data := []byte(generatedHeader + "\n//tgo:v2 \"model.tgo\" stale\n\npackage p\n")
	if metadata := parseGeneratedMetadata(data); metadata != nil {
		t.Fatalf("metadata: %#v", metadata)
	}
}

func TestReadTGoSourceUsesPassReader(t *testing.T) {
	called := false
	pass := &analysis.Pass{
		OtherFiles: nil,
		ReadFile: func(name string) ([]byte, error) {
			called = true
			return []byte(name), nil
		},
	}
	data, err := readTGoSource(pass, "/work/model.tgo")
	if err != nil {
		t.Fatal(err)
	}
	if !called || string(data) != "/work/model.tgo" {
		t.Fatalf("reader called=%v data=%q", called, data)
	}
	if len(pass.OtherFiles) != 1 || pass.OtherFiles[0] != "/work/model.tgo" {
		t.Fatalf("other files: %v", pass.OtherFiles)
	}
}

func TestSourceMatchesGenerated(t *testing.T) {
	tests := []struct {
		source    string
		generated string
		want      bool
	}{
		{source: "model.tgo", generated: "/work/model_tgo.go", want: true},
		{
			source: "model_linux.tgo", generated: "/work/model_tgo_linux.go", want: true,
		},
		{
			source:    "foo_tgo_bar_linux.tgo",
			generated: "/work/foo_tgo_bar_tgo_linux.go",
			want:      true,
		},
		{source: "foo_hack.tgo", generated: "/work/foo_tgo_hack.go", want: false},
	}
	for _, test := range tests {
		got := sourceMatchesGenerated(test.source, test.generated)
		if got != test.want {
			t.Errorf("mapping %s -> %s: got %v", test.source, test.generated, got)
		}
	}
}

func TestMissingGeneratedDeclarationDiagnostic(t *testing.T) {
	var message string
	pass := &analysis.Pass{
		Report: func(diagnostic analysis.Diagnostic) {
			message = diagnostic.Message
		},
		ExportPackageFact: func(analysis.Fact) {},
	}
	checker := &checker{pass: pass}
	file := &ast.File{Package: token.Pos(1)}
	data := []byte("package sample\ntype Missing struct {}\n")
	files := token.NewFileSet()
	source, err := syntax.ParseFile(files, "model.tgo", data, syntax.AllErrors)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	checker.checkSourceDeclaration(
		file, "model.tgo", source, source.Declarations[0],
		files.File(source.Package), data,
	)
	want := "generated tgo output for Missing does not match model.tgo"
	if message != want {
		t.Fatalf("diagnostic: %q", message)
	}
}

func TestCheckedSourceDeclarationFactRoundTrip(t *testing.T) {
	t.Parallel()
	models := sourceModelShapes(t)
	if sourceModelName(models[0]) != "Count" {
		t.Fatalf("checked source model: %#v", models[0])
	}
	checked := *models[0]
	switch checked.TgoTag() {
	case 1:
		shape := checked.TgoChecked()
		if shape.Base != "int" {
			t.Fatalf("checked base: %q", shape.Base)
		}
		assertSourceModelFactRoundTrip(t, shape.Fact, checkedModelWire, "Count", nil)
	case 2, 3:
		t.Fatal("checked source has a different variant")
		return
	default:
		panic("invalid sourceModel variant")
	}
}

func TestEnumSourceDeclarationFactRoundTrip(t *testing.T) {
	t.Parallel()
	models := sourceModelShapes(t)
	if sourceModelName(models[1]) != "Event" {
		t.Fatalf("enum source model: %#v", models[1])
	}
	enum := *models[1]
	switch enum.TgoTag() {
	case 1, 3:
		t.Fatal("enum source has a different variant")
		return
	case 2:
		shape := enum.TgoEnum()
		wantVariants := []string{"Started", "Stopped"}
		if len(shape.Variants) != 2 ||
			shape.Variants[0].name != wantVariants[0] ||
			shape.Variants[1].name != wantVariants[1] {
			t.Fatalf("enum variants: %#v", shape.Variants)
		}
		assertSourceModelFactRoundTrip(
			t, shape.Fact, enumModelWire, "Event", wantVariants,
		)
	default:
		panic("invalid sourceModel variant")
	}
}

func TestStructSourceDeclaration(t *testing.T) {
	t.Parallel()
	models := sourceModelShapes(t)
	if sourceModelName(models[2]) != "Options" {
		t.Fatalf("struct source model: %#v", models[2])
	}
	structure := *models[2]
	switch structure.TgoTag() {
	case 1, 2:
		t.Fatal("struct source has a different variant")
		return
	case 3:
		shape := structure.TgoStruct()
		if len(shape.Fields) != 1 || shape.Fields[0].name != "Limit" ||
			shape.Fields[0].typeExpression != "int" {
			t.Fatalf("struct fields: %#v", shape.Fields)
		}
	default:
		panic("invalid sourceModel variant")
	}
	if sourceModelFact(&structure) != nil {
		t.Fatal("struct source model has a fact")
	}
}

func sourceModelShapes(t *testing.T) []*sourceModel {
	t.Helper()
	data := []byte(`package sample

type Count int where value > 0

type Event enum {
	Started struct { ID string }
	Stopped struct { Reason string }
}

type Options struct { Limit int }
`)
	files := token.NewFileSet()
	file, err := syntax.ParseFile(files, "model.tgo", data, syntax.AllErrors)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	tokenFile := files.File(file.Package)
	models := make([]*sourceModel, 0, len(file.Declarations))
	for _, declaration := range file.Declarations {
		model := sourceDeclaration(
			declaration, "example.com/sample", file, tokenFile, data,
		)
		if model != nil {
			models = append(models, model)
		}
	}
	if len(models) != 3 {
		t.Fatalf("source models: %d", len(models))
	}
	return models
}

func assertSourceModelFactRoundTrip(
	t *testing.T,
	fact *model,
	wantKind uint8,
	wantName string,
	wantVariants []string,
) {
	t.Helper()
	wire := encodeModelFact(fact)
	if wire == nil || wire.Kind != wantKind || wire.Package != "example.com/sample" ||
		wire.Name != wantName || len(wire.Variants) != len(wantVariants) {
		t.Fatalf("wire fact: %#v", wire)
	}
	for index := range wantVariants {
		if wire.Variants[index] != wantVariants[index] {
			t.Fatalf("wire variants: %v", wire.Variants)
		}
	}
	decoded := decodeModelFact(wire, "example.com/sample")
	if !sameModelFact(fact, decoded) {
		t.Fatalf("fact round trip: %#v -> %#v", fact, decoded)
	}
}
