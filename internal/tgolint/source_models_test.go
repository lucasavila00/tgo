package tgolint

import (
	"bytes"
	"go/ast"
	"go/token"
	"testing"

	"tgo/syntax"

	"golang.org/x/tools/go/analysis"
)

func TestParseGeneratedMetadataUsesOnlyFixedHeaderLine(t *testing.T) {
	digest := bytes.Repeat([]byte("01"), 32)
	body := []byte("package p\n\n//tgo:v1 this is an ordinary source comment\n")
	data := append([]byte(generatedHeader+"\n//tgo:v1 \"model.tgo\" "), digest...)
	data = append(data, []byte("\n\n")...)
	data = append(data, body...)
	metadata := parseGeneratedMetadata(data)
	if metadata == nil || metadata.source != "model.tgo" ||
		!bytes.Equal(metadata.body, body) {
		t.Fatalf("metadata: %#v", metadata)
	}
}

func TestParseGeneratedMetadataRejectsUppercaseDigest(t *testing.T) {
	digest := bytes.Repeat([]byte("AB"), 32)
	data := append([]byte(generatedHeader+"\n//tgo:v1 \"model.tgo\" "), digest...)
	data = append(data, []byte("\n\npackage p\n")...)
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
	declaration := &syntax.StructDecl{
		Span: syntax.Span{Start: token.Pos(2), Stop: token.Pos(10)},
		Name: &ast.Ident{NamePos: token.Pos(7), Name: "Missing"},
	}
	checker.checkSourceDeclaration(file, "model.tgo", declaration)
	want := "generated tgo output for Missing does not match model.tgo"
	if message != want {
		t.Fatalf("diagnostic: %q", message)
	}
}
