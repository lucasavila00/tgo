package syntax_test

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"testing"

	"tgo/syntax"
)

func parseFile(t *testing.T, source string) (*token.FileSet, *syntax.File) {
	t.Helper()
	set := token.NewFileSet()
	file, err := syntax.ParseFile(
		set,
		"sample.tgo",
		[]byte(source),
		syntax.ParseComments|syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	return set, file
}

//nolint:cyclop // This test checks each extension type in one source file.
func TestParseExtensionsAndPositions(t *testing.T) {
	set, file := parseFile(t, `package sample

// State is a model.
type State enum {
	// On carries a name.
	On struct { Name string = "on" }
	Off struct{}
}

type Count int where value > 0

type Request struct {
	ID string
	Tags []string = []string{}
}

func label(value State) string {
	match value {
	case On(on):
		return on.Name
	case Off(_):
		return "off"
	}
}

var request = Request{ID: "one", ..default}
`)
	var enum *syntax.EnumDecl
	var checked *syntax.CheckedDecl
	var structure *syntax.StructDecl
	var match *syntax.MatchStmt
	var marker *syntax.DefaultMarker
	for _, extension := range syntax.Extensions(file) {
		switch node := extension.(type) {
		case *syntax.EnumDecl:
			enum = node
		case *syntax.CheckedDecl:
			checked = node
		case *syntax.StructDecl:
			structure = node
		case *syntax.MatchStmt:
			match = node
		case *syntax.DefaultMarker:
			marker = node
		}
	}
	if enum == nil || checked == nil || structure == nil || match == nil || marker == nil {
		t.Fatalf("missing extension: %#v", syntax.Extensions(file))
	}
	if got := set.Position(enum.Enum); got.Line != 4 || got.Column != 12 {
		t.Fatalf("enum position: %v", got)
	}
	if got := set.Position(checked.Where); got.Line != 10 || got.Column != 16 {
		t.Fatalf("where position: %v", got)
	}
	if got := set.Position(match.Match); got.Line != 18 || got.Column != 2 {
		t.Fatalf("match position: %v", got)
	}
	if got := set.Position(marker.Pos()); got.Line != 26 || got.Column != 34 {
		t.Fatalf("default position: %v", got)
	}
	if len(enum.Variants) != 2 || len(enum.Variants[0].Fields) != 1 {
		t.Fatalf("enum shape: %#v", enum)
	}
	if enum.Variants[0].Fields[0].Default == nil {
		t.Fatal("variant field default is missing")
	}
	if len(match.Cases) != 2 || match.Cases[0].Binding.Name != "on" {
		t.Fatalf("match shape: %#v", match)
	}
	if len(syntax.AttachedComments(file, enum)) == 0 {
		t.Fatal("enum comment is not attached")
	}
}

func TestContextualKeywordsStayGoIdentifiers(t *testing.T) {
	_, file := parseFile(t, `package sample

type enum int
type where int

func match[T any](value T) T { return value }

func use() enum { return match[enum](1) }
`)
	if len(syntax.Extensions(file)) != 0 {
		t.Fatalf("contextual names became extensions: %#v", syntax.Extensions(file))
	}
	if len(file.Decls) != 4 {
		t.Fatalf("ordinary declarations: %d", len(file.Decls))
	}
}

func TestErrorPropagationExpression(t *testing.T) {
	set, file := parseFile(t, `package sample

func load() (int, error) { return 1, nil }

func use() (int, error) {
	value := (load())!
	other := load()!
	if !ready(value) { return 0, nil }
	return value + other, nil
}

func ready(int) bool { return true }
`)
	propagations := []*syntax.PropagateExpr(nil)
	for _, extension := range syntax.Extensions(file) {
		if node, ok := extension.(*syntax.PropagateExpr); ok {
			propagations = append(propagations, node)
		}
	}
	if len(propagations) != 2 {
		t.Fatalf("propagation expressions: %d", len(propagations))
	}
	propagation := propagations[0]
	if propagation == nil || propagation.Call == nil {
		t.Fatal("propagation expression is missing")
	}
	if got := set.Position(propagation.Bang); got.Line != 6 || got.Column != 19 {
		t.Fatalf("bang position: %v", got)
	}
	if _, ok := syntax.Parent(file, propagation).(*ast.AssignStmt); !ok {
		t.Fatalf("propagation parent: %T", syntax.Parent(file, propagation))
	}
	if syntax.Parent(file, propagation.Expression) != propagation {
		t.Fatalf("expression parent: %T", syntax.Parent(file, propagation.Expression))
	}
	if len(syntax.Children(file, propagation)) != 1 {
		t.Fatalf("propagation children: %d", len(syntax.Children(file, propagation)))
	}
}

func TestNestedMatchesAndTraversal(t *testing.T) {
	_, file := parseFile(t, `package sample

type State enum {
	On struct{}
	Off struct{}
}

func nested(outer State, inner State) {
Done:
	match outer {
	case On(_):
		if true {
			match inner {
			case On(_): break Done
			case Off(_): return
			}
		}
	case Off(_): return
	}
}
`)
	matchCount := 0
	caseCount := 0
	goReturnCount := 0
	syntax.Inspect(file, func(node syntax.Node) bool {
		switch node.(type) {
		case *syntax.MatchStmt:
			matchCount++
		case *syntax.MatchCase:
			caseCount++
		case *ast.ReturnStmt:
			goReturnCount++
		}
		return true
	})
	if matchCount != 2 || caseCount != 4 || goReturnCount != 2 {
		t.Fatalf(
			"walk counts: matches=%d cases=%d returns=%d",
			matchCount, caseCount, goReturnCount,
		)
	}
	for _, extension := range syntax.Extensions(file) {
		if syntax.Parent(file, extension) == nil {
			t.Fatalf("extension has no parent: %T", extension)
		}
	}
}

func TestDefaultMarkerDoesNotExposeProjection(t *testing.T) {
	_, file := parseFile(t, `package sample

type Request struct { Name string = "default" }

var request = Request{..default}
`)
	artificial := false
	ast.Inspect(file.GoFile(), func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if ok && literal.Value == "0" {
			artificial = true
		}
		return true
	})
	if artificial {
		t.Fatal("default projection escaped through GoFile")
	}
	if len(syntax.Extensions(file)) < 3 {
		t.Fatalf("extensions: %#v", syntax.Extensions(file))
	}
}

func TestSyntaxErrors(t *testing.T) {
	tests := []string{
		"package p\ntype E enum { Bad string }",
		"package p\ntype Q int where",
		"package p\nfunc f() { match value { case Bad(1): return } }",
		"package p\ntype S struct { Value int = }",
	}
	for _, source := range tests {
		set := token.NewFileSet()
		_, err := syntax.ParseFile(set, "bad.tgo", []byte(source), syntax.AllErrors)
		if err == nil {
			t.Fatalf("expected error for %q", source)
		}
	}
}

func TestAllErrorsReportsScannerErrors(t *testing.T) {
	source := []byte("package p\nvar first = \x00\nvar second = \x00\n")
	_, err := syntax.ParseFile(token.NewFileSet(), "bad.tgo", source, syntax.AllErrors)
	var list scanner.ErrorList
	if !errors.As(err, &list) || len(list) < 2 {
		t.Fatalf("scanner errors: %T %v", err, err)
	}
}

func TestOrdinaryGoASTParity(t *testing.T) {
	source := `// package comment
package sample

import "fmt"

type Pair[T any] struct { Left, Right T }

func String(value int) string {
	if value > 0 {
		return fmt.Sprint(value)
	}
	return "zero"
}
`
	set, file := parseFile(t, source)
	standardSet := token.NewFileSet()
	standard, err := parser.ParseFile(
		standardSet,
		"sample.tgo",
		source,
		parser.ParseComments|parser.SkipObjectResolution,
	)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := format.Node(&got, set, file.GoFile()); err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	if err := format.Node(&want, standardSet, standard); err != nil {
		t.Fatal(err)
	}
	if got.String() != want.String() {
		t.Fatalf("ordinary Go AST differs\ngot:\n%s\nwant:\n%s", got.String(), want.String())
	}
	if len(syntax.Extensions(file)) != 0 {
		t.Fatalf("ordinary Go has extensions: %#v", syntax.Extensions(file))
	}
}

func TestExactSpansAndCommentOrder(t *testing.T) {
	set, file := parseFile(t, `package sample

// before
type Choice enum { // body
	// first
	One struct{} // after
}
`)
	declaration := syntax.Extensions(file)[0].(*syntax.EnumDecl)
	start := set.Position(declaration.Pos())
	end := set.Position(declaration.End())
	if start.Line != 4 || start.Column != 1 || end.Line != 7 || end.Column != 2 {
		t.Fatalf("enum span: %v to %v", start, end)
	}
	want := []string{"// before", "// body", "// first", "// after"}
	got := []string(nil)
	for _, group := range file.Comments {
		for _, comment := range group.List {
			got = append(got, comment.Text)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("comment count: %d", len(got))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("comment %d: %q", index, got[index])
		}
	}
	comments := syntax.AttachedComments(file, declaration)
	if len(comments) == 0 || comments[0].List[0].Text != "// before" {
		t.Fatalf("attached comments: %#v", comments)
	}
}

func TestContextualWhereKeepsCompleteGoTypes(t *testing.T) {
	_, file := parseFile(t, `package sample

type where int
type F func() where
`)
	if len(syntax.Extensions(file)) != 0 {
		t.Fatalf("Go types became tgo declarations: %#v", syntax.Extensions(file))
	}
}

func TestContextualMatchKeepsCompleteGoStatements(t *testing.T) {
	_, file := parseFile(t, `package sample

type T = struct{}
type C = chan T

func use(channel C) {
	match := T{}
	_ = match
	match += T{}
	match <- C{}
}
`)
	for _, extension := range syntax.Extensions(file) {
		if _, ok := extension.(*syntax.MatchStmt); ok {
			t.Fatalf("Go statement became match: %#v", extension)
		}
	}
}

func TestEmptyMatchBeforeBlockEnd(t *testing.T) {
	_, file := parseFile(t, `package sample
func use(value any) { match value {} }
`)
	if len(syntax.Extensions(file)) != 1 {
		t.Fatalf("extensions: %#v", syntax.Extensions(file))
	}
	if _, ok := syntax.Extensions(file)[0].(*syntax.MatchStmt); !ok {
		t.Fatalf("extension type: %T", syntax.Extensions(file)[0])
	}
}

func TestMultilineMatchSubject(t *testing.T) {
	_, file := parseFile(t, `package sample
func use(value any) {
	match
		value {
	case One(item): _ = item
	}
}
`)
	if len(syntax.Extensions(file)) != 2 {
		t.Fatalf("extensions: %#v", syntax.Extensions(file))
	}
}

func TestContextualMatchBeforeBlockEnd(t *testing.T) {
	_, file := parseFile(t, `package sample
type T = struct{}
type C = chan T
func assign() { match := T{} }
func add(match T) { match += T{} }
func send(match C) { match <- T{} }
`)
	if len(syntax.Extensions(file)) != 0 {
		t.Fatalf("Go statements became extensions: %#v", syntax.Extensions(file))
	}
}

func TestDefaultMarkerNeedsCompositeLiteral(t *testing.T) {
	set := token.NewFileSet()
	file, err := syntax.ParseFile(
		set,
		"bad.tgo",
		[]byte("package p\nfunc f() { _ = ..default }\n"),
		syntax.AllErrors,
	)
	if err == nil || file != nil {
		t.Fatalf("invalid marker result: file=%#v error=%v", file, err)
	}
}

func TestTraversalVisitsEachNodeOnce(t *testing.T) {
	_, file := parseFile(t, `package sample

// enum doc
type State enum { On struct{}; Off struct{} }

func use(outer State, inner State) {
	match outer {
	case On(_):
		match inner {
		case On(_): return
		case Off(_): return
		}
	case Off(_): return
	}
}
`)
	seen := make(map[syntax.Node]bool)
	syntax.Inspect(file, func(node syntax.Node) bool {
		if seen[node] {
			t.Fatalf("node visited twice: %T at %v", node, node.Pos())
		}
		seen[node] = true
		for _, child := range syntax.Children(file, node) {
			if syntax.Parent(file, child) != node {
				t.Fatalf("parent mismatch: %T -> %T", node, child)
			}
		}
		return true
	})
}

func TestFragmentErrorsUseOriginalLocation(t *testing.T) {
	tests := []struct {
		source string
		line   int
	}{
		{source: "package p\ntype S struct { Value [] }\n", line: 2},
		{source: "package p\ntype Q int where value +\n", line: 2},
		{source: "package p\nfunc f() { match value { case One(_): return + } }\n", line: 2},
	}
	for _, test := range tests {
		set := token.NewFileSet()
		_, err := syntax.ParseFile(set, "original.tgo", []byte(test.source), syntax.AllErrors)
		if err == nil {
			t.Fatalf("expected error for %q", test.source)
		}
		if !strings.Contains(err.Error(), "original.tgo:"+strconv.Itoa(test.line)+":") {
			t.Fatalf("unmapped error: %v", err)
		}
		if strings.Contains(err.Error(), "(field)") ||
			strings.Contains(err.Error(), "(expression)") ||
			strings.Contains(err.Error(), "(statements)") {
			t.Fatalf("wrapper name escaped: %v", err)
		}
	}
}

func TestComposedDefaultMarkers(t *testing.T) {
	_, file := parseFile(t, `package sample

type Inner struct { Value int = 1 }
type Request struct { Item Inner = Inner{..default} }
type Event enum { One struct{} }

func use() {
	match makeEvent(Request{..default}) {
	case One(value):
		_ = Request{..default}
	}
}
`)
	markers := 0
	for _, extension := range syntax.Extensions(file) {
		if _, ok := extension.(*syntax.DefaultMarker); ok {
			markers++
		}
	}
	if markers != 3 {
		t.Fatalf("default marker count: %d", markers)
	}
	ast.Inspect(file.GoFile(), func(node ast.Node) bool {
		if literal, ok := node.(*ast.BasicLit); ok && literal.Value == "0" {
			t.Fatalf("artificial literal at %v", literal.Pos())
		}
		return true
	})
}

func TestNestedMatchInSubject(t *testing.T) {
	_, file := parseFile(t, `package sample

type E enum { A struct{} }

func use(e E) {
	match func() E {
		match e { case A(a): return e }
		return e
	}() { case A(a): _ = a }
}
`)
	matches := 0
	syntax.Inspect(file, func(node syntax.Node) bool {
		if _, ok := node.(*syntax.MatchStmt); ok {
			matches++
		}
		if _, artificial := node.(*ast.EmptyStmt); artificial {
			t.Fatalf("artificial statement at %v", node.Pos())
		}
		for _, child := range syntax.Children(file, node) {
			if syntax.Parent(file, child) != node {
				t.Fatalf("parent mismatch: %T -> %T", node, child)
			}
		}
		return true
	})
	if matches != 2 {
		t.Fatalf("match count: %d", matches)
	}
}

func TestLabelsAroundMatchUseSyntaxNodes(t *testing.T) {
	_, file := parseFile(t, `package sample

type E enum { A struct{} }

func use(e E) {
Outer: Inner: match e { case A(a): break Outer }
}
`)
	labels := []string(nil)
	syntax.Inspect(file, func(node syntax.Node) bool {
		if label, ok := node.(*syntax.LabeledStmt); ok {
			labels = append(labels, label.Label.Name)
		}
		return true
	})
	if len(labels) != 2 || labels[0] != "Outer" || labels[1] != "Inner" {
		t.Fatalf("labels: %v", labels)
	}
	ast.Inspect(file.GoFile(), func(node ast.Node) bool {
		if statement, ok := node.(*ast.EmptyStmt); ok && !statement.Implicit {
			t.Fatalf("artificial labeled statement at %v", statement.Pos())
		}
		return true
	})
}

func TestCommentOwnership(t *testing.T) {
	_, file := parseFile(t, `package sample

// declaration
type E enum { A struct{} }


// unattached
`)
	declaration := syntax.Extensions(file)[0].(*syntax.EnumDecl)
	attached := syntax.AttachedComments(file, declaration)
	if len(attached) != 1 || syntax.Parent(file, attached[0]) != declaration {
		t.Fatalf("attached owner: %T", syntax.Parent(file, attached[0]))
	}
	var loose *ast.CommentGroup
	for _, comment := range file.Comments {
		if comment.Text() == "unattached\n" {
			loose = comment
		}
	}
	if loose == nil || syntax.Parent(file, loose) != file {
		t.Fatalf("unattached owner: %T", syntax.Parent(file, loose))
	}
	visits := 0
	syntax.Inspect(file, func(node syntax.Node) bool {
		if node == loose {
			visits++
		}
		return true
	})
	if visits != 1 {
		t.Fatalf("unattached comment visits: %d", visits)
	}
}

func TestPackageCommentOwnership(t *testing.T) {
	_, file := parseFile(t, `// package sample documents sample.
package sample
`)
	if file.Doc == nil || syntax.Parent(file, file.Doc) != file {
		t.Fatalf("package comment owner: %T", syntax.Parent(file, file.Doc))
	}
	comments := syntax.AttachedComments(file, file)
	if len(comments) != 1 || comments[0] != file.Doc {
		t.Fatalf("package comments: %#v", comments)
	}
}

func TestGoCommentOwnershipParity(t *testing.T) {
	_, file := parseFile(t, `package sample
const Value = 1 // value
type Number int // number
`)
	want := make(map[string]syntax.Node)
	for _, declaration := range file.GoFile().Decls {
		general := declaration.(*ast.GenDecl)
		specification := general.Specs[0]
		switch specification.(type) {
		case *ast.ValueSpec:
			want["value\n"] = specification
		case *ast.TypeSpec:
			want["number\n"] = specification
		}
	}
	for _, comment := range file.Comments {
		owner := want[comment.Text()]
		if owner != nil && syntax.Parent(file, comment) != owner {
			t.Fatalf("comment %q owner: %T", comment.Text(), syntax.Parent(file, comment))
		}
	}
}

func TestVariantLiteralOfRejectsNilAndValueSelectors(t *testing.T) {
	if literal, ok := syntax.VariantLiteralOf(nil, nil, nil); ok || literal != nil {
		t.Fatalf("nil literal classified: %#v", literal)
	}
	receiver := ast.NewIdent("value")
	literal := &ast.CompositeLit{
		Type: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent("Variant")},
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{
		receiver: {Type: types.Typ[types.Int]},
	}}
	if result, ok := syntax.VariantLiteralOf(literal, info, func(types.Type) bool {
		return true
	}); ok || result != nil {
		t.Fatalf("value selector classified: %#v", result)
	}
}
