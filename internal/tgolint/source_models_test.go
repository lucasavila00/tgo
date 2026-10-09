package tgolint

import (
	"go/token"
	"testing"

	"tgo/pkg/syntax"

	"golang.org/x/tools/go/analysis"
)

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

func TestMissingGeneratedDeclarationDiagnostic(t *testing.T) {
	var message string
	pass := &analysis.Pass{
		Report: func(diagnostic analysis.Diagnostic) {
			message = diagnostic.Message
		},
		ExportPackageFact: func(analysis.Fact) {},
	}
	checker := &checker{pass: pass}
	data := []byte("package sample\ntype Missing struct {}\n")
	files := token.NewFileSet()
	generated, err := syntax.ParseGoFile(
		files, "model_tgo.go", []byte("package sample\n"), syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("ParseFile generated: %v", err)
	}
	source, err := syntax.ParseFile(files, "model.tgo", data, syntax.AllErrors)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	checker.checkSourceDeclaration(
		generated, "model.tgo", source, source.Declarations[0],
		files.File(source.Package), data,
	)
	want := "generated tgo output for Missing does not match model.tgo"
	if message != want {
		t.Fatalf("diagnostic: %q", message)
	}
}

func TestCheckedStructSourceDeclarationFactRoundTrip(t *testing.T) {
	t.Parallel()
	models := sourceModelShapes(t)
	if sourceModelName(models[0]) != "Count" {
		t.Fatalf("checked source model: %#v", models[0])
	}
	structure := *models[0]
	switch structure.Tag() {
	case sourceModelTagEnum:
		t.Fatal("checked source has a different variant")
	case sourceModelTagStruct:
		shape := structure.StructPayload()
		if len(shape.Fields) != 1 || shape.Fields[0].name != "value" {
			t.Fatalf("checked fields: %#v", shape.Fields)
		}
		assertSourceModelFactRoundTrip(t, shape.Fact, checkedModelWire, "Count", nil)
	default:
		panic(structure.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func TestEnumSourceDeclarationFactRoundTrip(t *testing.T) {
	t.Parallel()
	models := sourceModelShapes(t)
	if sourceModelName(models[1]) != "Event" {
		t.Fatalf("enum source model: %#v", models[1])
	}
	enum := *models[1]
	switch enum.Tag() {
	case sourceModelTagStruct:
		t.Fatal("enum source has a different variant")
	case sourceModelTagEnum:
		shape := enum.EnumPayload()
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
		panic(enum.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func TestStructSourceDeclaration(t *testing.T) {
	t.Parallel()
	models := sourceModelShapes(t)
	if sourceModelName(models[2]) != "Options" {
		t.Fatalf("struct source model: %#v", models[2])
	}
	structure := *models[2]
	switch structure.Tag() {
	case sourceModelTagEnum:
		t.Fatal("struct source has a different variant")
	case sourceModelTagStruct:
		shape := structure.StructPayload()
		if len(shape.Fields) != 1 || shape.Fields[0].name != "Limit" ||
			shape.Fields[0].typeExpression != "int" {
			t.Fatalf("struct fields: %#v", shape.Fields)
		}
	default:
		panic(structure.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
	if sourceModelFact(&structure) != nil {
		t.Fatal("struct source model has a fact")
	}
}

func sourceModelShapes(t *testing.T) []*sourceModel {
	t.Helper()
	data := []byte(`package sample

type Count struct { value int } checked

func (value Count) check() (Count, error) { return value, nil }

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
