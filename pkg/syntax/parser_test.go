package syntax_test

import (
	"go/token"
	"os"
	"reflect"
	"testing"

	"tgo/pkg/syntax"
)

func TestParseFileMarksSuccessfulReturns(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("testdata/success-return/valid.tgo")
	if err != nil {
		t.Fatal(err)
	}
	files := token.NewFileSet()
	file, err := syntax.ParseFile(
		files, "valid.tgo", source, syntax.ParseComments|syntax.AllErrors,
	)
	if err != nil {
		t.Fatal(err)
	}
	marked := 0
	ordinary := 0
	syntax.Inspect(file, func(node *syntax.Node) bool {
		statement, ok := syntax.StatementOf(node)
		if !ok {
			return true
		}
		returned := syntax.ReturnStatementOf(statement)
		if returned == nil {
			return true
		}
		if !returned.SuccessComma.IsValid() {
			ordinary++
			return true
		}
		marked++
		if source[files.File(returned.SuccessComma).Offset(returned.SuccessComma)] != ',' {
			t.Fatalf("success marker is not a comma")
		}
		if text := syntax.SourceText(file, returned.Span); text[len(text)-1] != ',' {
			t.Fatalf("return span = %q", text)
		}
		return true
	})
	if marked != 5 || ordinary != 1 {
		t.Fatalf("marked returns = %d, ordinary returns = %d", marked, ordinary)
	}
}

func TestParseFileMarksFailureReturns(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("testdata/failure-return/valid.tgo")
	if err != nil {
		t.Fatal(err)
	}
	files := token.NewFileSet()
	file, err := syntax.ParseFile(
		files, "valid.tgo", source, syntax.ParseComments|syntax.AllErrors,
	)
	if err != nil {
		t.Fatal(err)
	}
	counts := []int(nil)
	syntax.Inspect(file, func(node *syntax.Node) bool {
		statement, ok := syntax.StatementOf(node)
		if !ok {
			return true
		}
		returned := syntax.ReturnStatementOf(statement)
		if returned == nil || len(returned.FailureCommas) == 0 {
			return true
		}
		counts = append(counts, len(returned.FailureCommas))
		if len(returned.Results) != 1 {
			t.Fatalf("failure return has %d expressions", len(returned.Results))
		}
		for _, comma := range returned.FailureCommas {
			if source[files.File(comma).Offset(comma)] != ',' {
				t.Fatal("failure marker is not a comma")
			}
		}
		return true
	})
	if !reflect.DeepEqual(counts, []int{1, 2, 3}) {
		t.Fatalf("failure comma counts = %v", counts)
	}
}

func TestParseFileRejectsInvalidFailureReturns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "empty", want: "empty.tgo:4:9: failure return needs one error expression"},
		{
			name: "multiple",
			want: "multiple.tgo:4:14: failure return needs exactly one error expression",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, err := os.ReadFile("testdata/failure-return/" + test.name + ".tgo")
			if err != nil {
				t.Fatal(err)
			}
			_, err = syntax.ParseFile(
				token.NewFileSet(), test.name+".tgo", source, syntax.AllErrors,
			)
			if err == nil || err.Error() != test.want {
				t.Fatalf("error = %q, want %q", err, test.want)
			}
		})
	}
}

func TestParseFileRejectsRemovedWhereDeclaration(t *testing.T) {
	t.Parallel()
	_, err := syntax.ParseFile(
		token.NewFileSet(), "removed.tgo",
		[]byte("package sample\n\ntype Port int where value > 0\n"),
		syntax.AllErrors,
	)
	if err == nil {
		t.Fatal("ParseFile accepted a removed where declaration")
	}
}

func TestParseFileConvertsAllPublicForms(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample

import alias "example.invalid/value"

const constant = 1
var global int
type Alias = int
type Generic[T any] []T
type Required = %int

type Result enum {
	OK struct { Value int }
	Error struct { Message string }
}
type Options struct { Limit int = 10 }
type Port struct { value int } checked

func work[T any](receiver int, values ...T) (result int) {
	var local int
	var required %int
	;
start:
	local++
	channel := make(chan int)
	channel <- local
	local--
	println(local)
	local, global = global, local
	go println(local)
	defer println(local)
	{
		local = -local + 2
	}
	if next := local + 1; next > 0 {
		local = next
	} else {
		local = 0
	}
	switch local {
	case 1, 2:
		local++
	default:
		local--
	}
	var unknown any = local
	switch value := unknown.(type) {
	case int:
		local = value
	}
	select {
	case channel <- local:
		local++
	default:
	}
	for index := 0; index < 1; index++ {
		continue
	}
	for index, value := range []int{1, 2} {
		local += index + value
	}
	array := [2]int{0: 1}
	slice := array[:1:2]
	_ = struct{ Name string }{Name: "x"}
	_ = func(input int) int { return input }(local)
	_ = (local)
	_ = alias.Value
	_ = slice[0]
	_ = work[int]
	_ = work[int, string]
	_ = unknown.(int)
	_ = &local
	_ = *(&local)
	_ = required
	_ = local % 2
	_ = map[string]int{"x": 1}
	_ = interface{ String() string }(nil)
	_ = chan<- int(channel)
	_ = (<-chan int)(channel)
	_ = Options{Limit: 1, ..default}
	_ = load()!
	_ = []int{for _, value := range []int{1, 2} {
		if value > 0 { value }
	}}
	if local == 0 {
		goto start
	}
	return local
}

func load() (int, error) { return 0, nil }
`)
	files := token.NewFileSet()
	file, err := syntax.ParseFile(
		files, "forms.tgo", source, syntax.ParseComments|syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	expressions := make(map[string]bool)
	statements := make(map[string]bool)
	declarations := make(map[string]bool)
	specifications := make(map[string]bool)
	syntax.Inspect(file, func(node *syntax.Node) bool {
		if value, ok := syntax.ExpressionOf(node); ok {
			expressions[syntax.ExpressionKind(value)] = true
		}
		if value, ok := syntax.StatementOf(node); ok {
			statements[syntax.StatementKind(value)] = true
		}
		if value, ok := syntax.DeclarationOf(node); ok {
			declarations[syntax.DeclarationKind(value)] = true
		}
		if value, ok := syntax.SpecificationOf(node); ok {
			specifications[syntax.SpecificationKind(value)] = true
		}
		return true
	})

	requireKinds(t, expressions, []string{
		"Identifier", "Ellipsis", "BasicLiteral", "FunctionLiteral",
		"CompositeLiteral", "Parenthesized", "Selector", "Index", "IndexList",
		"Slice", "TypeAssertion", "Call", "Star", "NonNilPointer", "Unary", "Binary",
		"KeyValue", "ArrayType", "StructType", "FunctionType", "InterfaceType",
		"MapType", "ChannelType", "Default", "Propagation", "Comprehension",
	})
	requireKinds(t, statements, []string{
		"Declaration", "Empty", "Labeled", "Expression", "Send", "Increment",
		"Assignment", "Go", "Defer", "Return", "Branch", "Block", "If", "Case",
		"Switch", "TypeSwitch", "Communication", "Select", "For", "Range",
	})
	requireKinds(t, declarations, []string{
		"General", "Function", "Enum", "Struct",
	})
	requireKinds(t, specifications, []string{"Import", "Value", "Type"})
}

func TestParseFileMarksOnlySwitchExhaustiveClause(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample

type holder struct { exhaustive int }

func inspect(value int) {
	_ = holder{exhaustive: 1}
	switch value {
	case 1:
	exhaustive:
	}
}

func TestParseGoFileKeepsGoStructTypeSpecification(t *testing.T) {
	t.Parallel()
	source := []byte("package sample\n\ntype Value struct { Field int }\n")
	file, err := syntax.ParseGoFile(
		token.NewFileSet(), "value.go", source, syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("ParseGoFile: %v", err)
	}
	if len(file.Declarations) != 1 {
		t.Fatalf("declarations = %d, want 1", len(file.Declarations))
	}
	general := syntax.GeneralDeclarationOf(file.Declarations[0])
	if general == nil || len(general.Specs) != 1 {
		t.Fatalf("general declaration: %#v", general)
	}
	specification := syntax.TypeSpecificationOf(general.Specs[0])
	if specification == nil || specification.Name.Name != "Value" {
		t.Fatalf("type specification: %#v", specification)
	}
	if got := syntax.SourceText(file, syntax.Span{
		Start: syntax.ExpressionPosition(specification.Type),
		Stop:  syntax.ExpressionEnd(specification.Type),
	}); got != "struct { Field int }" {
		t.Fatalf("source text = %q", got)
	}
}

`)
	file, err := syntax.ParseFile(
		token.NewFileSet(), "exhaustive.tgo", source, syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	marked := 0
	syntax.Inspect(file, func(node *syntax.Node) bool {
		statement, ok := syntax.StatementOf(node)
		if !ok || statement.Tag() != syntax.StatementTagCase {
			return true
		}
		if statement.CasePayload().Value.Exhaustive.IsValid() {
			marked++
		}
		return true
	})
	if marked != 1 {
		t.Fatalf("marked exhaustive clauses = %d, want 1", marked)
	}
}

func TestParseFileMarksCheckedStructAndLiteralPropagation(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample

type Port struct { number int } checked

func makePort(number int) (Port, error) {
	return Port{number: number}!
}
`)
	file, err := syntax.ParseFile(
		token.NewFileSet(), "checked.tgo", source, syntax.AllErrors,
	)
	if err != nil {
		t.Fatal(err)
	}
	structure, ok := syntax.StructDeclarationOf(file.Declarations[0])
	if !ok || structure.Checked == token.NoPos {
		t.Fatal("checked struct marker is missing")
	}
	propagations := 0
	syntax.Inspect(file, func(node *syntax.Node) bool {
		propagation, ok := syntax.PropagationExpressionOf(node)
		if !ok {
			return true
		}
		propagations++
		if syntax.CompositeLiteralOf(propagation.Expression) == nil {
			t.Fatal("propagation does not contain the checked struct literal")
		}
		return true
	})
	if propagations != 1 {
		t.Fatalf("propagations = %d, want 1", propagations)
	}
}

func TestPublicASTDoesNotExposeGoAST(t *testing.T) {
	t.Parallel()
	seen := make(map[reflect.Type]bool)
	var inspect func(reflect.Type)
	inspect = func(value reflect.Type) {
		if value == nil || seen[value] {
			return
		}
		seen[value] = true
		if value.PkgPath() == "go/ast" {
			t.Fatalf("public AST exposes %s", value)
		}
		switch value.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			inspect(value.Elem())
		case reflect.Map:
			inspect(value.Key())
			inspect(value.Elem())
		case reflect.Struct:
			for index := 0; index < value.NumField(); index++ {
				field := value.Field(index)
				if field.IsExported() {
					inspect(field.Type)
				}
			}
		}
	}
	inspect(reflect.TypeFor[syntax.File]())
	inspect(reflect.TypeFor[syntax.Expression]())
	inspect(reflect.TypeFor[syntax.Statement]())
	inspect(reflect.TypeFor[syntax.Declaration]())
	inspect(reflect.TypeFor[syntax.Specification]())
}

func TestParentAndChildrenUseTGoNodes(t *testing.T) {
	t.Parallel()
	file, err := syntax.ParseFile(
		token.NewFileSet(),
		"small.tgo",
		[]byte("package small\nvar value = 1\n"),
		syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	count := 0
	syntax.Inspect(file, func(node *syntax.Node) bool {
		count++
		if syntax.Parent(file, node) == nil && count != 1 {
			t.Fatalf("node %d has no parent", count)
		}
		return true
	})
	if count < 5 {
		t.Fatalf("visited %d nodes", count)
	}
}

func TestParseFilePropagationKeepsParserErrorPosition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "one result",
			source: "package sample\n" +
				"type Result enum {\n" +
				"\tBad struct { Value (int] }\n" +
				"}\n",
			want: "bad.tgo:p.closeToken: 3:25: unmatched ]",
		},
		{
			name: "two results",
			source: "package sample\n" +
				"type Result enum {\n" +
				"\tBad int\n" +
				"}\n",
			want: "bad.tgo:p.variant: 3:2: variant needs Name struct { fields }",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := syntax.ParseFile(
				token.NewFileSet(), "bad.tgo", []byte(test.source), syntax.AllErrors,
			)
			if err == nil {
				t.Fatal("ParseFile succeeded")
			}
			if got := err.Error(); got != test.want {
				t.Fatalf("error = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseFileTransparentPropagation(t *testing.T) {
	t.Parallel()
	source := []byte("package sample\nfunc load() (int, error) { return 0, nil }\n" +
		"func use() (int, error) { return load()!!, nil }\n")
	files := token.NewFileSet()
	file, err := syntax.ParseFile(files, "transparent.tgo", source, syntax.AllErrors)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var propagation *syntax.PropagationExpression
	for _, extension := range syntax.Extensions(file) {
		if value, ok := syntax.PropagationExpressionOf(extension); ok {
			propagation = value
		}
	}
	if propagation == nil {
		t.Fatal("transparent propagation is absent")
	}
	first := files.Position(propagation.Bang)
	second := files.Position(propagation.SecondBang)
	end := files.Position(propagation.Stop)
	if first.Line != 3 || first.Column != 40 ||
		second.Line != 3 || second.Column != 41 ||
		end.Line != 3 || end.Column != 42 {
		t.Fatalf("positions = %v, %v, %v", first, second, end)
	}
}

func TestParseFileTransparentPropagationNeedsAdjacentMarks(t *testing.T) {
	t.Parallel()
	source := []byte("package sample\nfunc load() (int, error) { return 0, nil }\n" +
		"func use() (int, error) { return load()! !, nil }\n")
	_, err := syntax.ParseFile(
		token.NewFileSet(), "spaced.tgo", source, syntax.AllErrors,
	)
	if err == nil {
		t.Fatal("ParseFile accepted spaced propagation marks")
	}
}

func TestComprehensionDiscoveryInsideLeadingLoop(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample
func collect(values []int) {
	for range values {
		_ = []int{for _, value := range values { value }}
	}
}
`)
	file, err := syntax.ParseFile(
		token.NewFileSet(), "nested.tgo", source, syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	count := 0
	for _, extension := range syntax.Extensions(file) {
		if _, ok := syntax.ComprehensionExpressionOf(extension); ok {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("found %d comprehensions", count)
	}
}

func TestParseFileComprehensionClausesKeepOrderAndPositions(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample
func collect(matrix [][]int) []int {
return []int{for row, values := range matrix {
for _, value := range values {
if value > row { value }
}
}}
}
`)
	files := token.NewFileSet()
	file, err := syntax.ParseFile(
		files, "clauses.tgo", source, syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var comprehension *syntax.ComprehensionExpression
	for _, extension := range syntax.Extensions(file) {
		if value, ok := syntax.ComprehensionExpressionOf(extension); ok {
			comprehension = value
		}
	}
	if comprehension == nil {
		t.Fatal("comprehension is absent")
	}
	if len(comprehension.Clauses) != 3 {
		t.Fatalf("clause count = %d, want 3", len(comprehension.Clauses))
	}

	outer, ok := syntax.ComprehensionRangeClauseOf(&comprehension.Clauses[0])
	if !ok {
		t.Fatal("first clause is not a range")
	}
	inner, ok := syntax.ComprehensionRangeClauseOf(&comprehension.Clauses[1])
	if !ok {
		t.Fatal("second clause is not a range")
	}
	filter, ok := syntax.ComprehensionFilterClauseOf(&comprehension.Clauses[2])
	if !ok {
		t.Fatal("third clause is not a filter")
	}
	if len(outer.Bindings) != 2 || outer.Bindings[0].Name != "row" ||
		outer.Bindings[1].Name != "values" {
		t.Fatalf("outer bindings = %#v", outer.Bindings)
	}
	if len(inner.Bindings) != 2 || inner.Bindings[0].Name != "_" ||
		inner.Bindings[1].Name != "value" {
		t.Fatalf("inner bindings = %#v", inner.Bindings)
	}

	positions := []struct {
		name string
		got  token.Pos
		line int
		col  int
	}{
		{name: "outer for", got: outer.For, line: 3, col: 14},
		{name: "outer start", got: outer.Start, line: 3, col: 14},
		{name: "outer stop", got: outer.Stop, line: 7, col: 2},
		{name: "outer define", got: outer.Define, line: 3, col: 30},
		{name: "outer range", got: outer.Range, line: 3, col: 33},
		{name: "outer open", got: outer.Lbrace, line: 3, col: 46},
		{name: "outer close", got: outer.Rbrace, line: 7, col: 1},
		{name: "inner for", got: inner.For, line: 4, col: 1},
		{name: "inner start", got: inner.Start, line: 4, col: 1},
		{name: "inner stop", got: inner.Stop, line: 6, col: 2},
		{name: "inner open", got: inner.Lbrace, line: 4, col: 30},
		{name: "inner close", got: inner.Rbrace, line: 6, col: 1},
		{name: "filter if", got: filter.If, line: 5, col: 1},
		{name: "filter start", got: filter.Start, line: 5, col: 1},
		{name: "filter stop", got: filter.Stop, line: 5, col: 25},
		{name: "filter open", got: filter.Lbrace, line: 5, col: 16},
		{name: "filter close", got: filter.Rbrace, line: 5, col: 24},
	}
	for _, item := range positions {
		position := files.Position(item.got)
		if position.Line != item.line || position.Column != item.col {
			t.Errorf(
				"%s position = %d:%d, want %d:%d",
				item.name, position.Line, position.Column, item.line, item.col,
			)
		}
	}
}

func TestParseFileRejectsComprehensionRangeAfterFilter(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample
func collect(values []int) []int {
return []int{for _, value := range values {
if value > 0 {
for _, next := range values { next }
}
}}
}
`)
	_, err := syntax.ParseFile(
		token.NewFileSet(), "order.tgo", source, syntax.AllErrors,
	)
	want := "order.tgo:5:1: comprehension ranges must precede the filter"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}

func requireKinds(t *testing.T, found map[string]bool, expected []string) {
	t.Helper()
	missing := []string(nil)
	for _, kind := range expected {
		if !found[kind] {
			missing = append(missing, kind)
		}
	}
	if len(missing) != 0 {
		t.Fatalf("missing kinds %v; found %v", missing, found)
	}
}

func TestEnumJSONTags(t *testing.T) {
	source := []byte("package sample\n" +
		"type Event enum `json:\"adjacent,tag=type,content=data\"`\n{\n" +
		" Created struct { ID string `json:\"id\"` } `json:\"created\"` // variant\n" +
		" Empty struct {}\n}\n")
	files := token.NewFileSet()
	tree, err := syntax.ParseFile(files, "sample.tgo", source, syntax.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	enum, ok := syntax.EnumDeclarationOf(tree.Declarations[0])
	if !ok || enum.Tag == nil || enum.Tag.Value != "`json:\"adjacent,tag=type,content=data\"`" {
		t.Fatal("enum tag was not parsed")
	}
	variant := enum.Variants[0]
	if variant.Tag == nil || variant.Tag.Value != "`json:\"created\"`" {
		t.Fatal("variant tag was not parsed")
	}
	if variant.Stop != variant.Tag.Stop {
		t.Fatal("variant span does not include its tag")
	}
	if enum.Variants[1].Tag != nil {
		t.Fatal("untagged variant has a tag")
	}
	positions := map[token.Pos]bool{
		enum.Tag.Start:                    true,
		variant.Tag.Start:                 true,
		variant.Fields[0].Field.Tag.Start: true,
	}
	tags := 0
	syntax.Inspect(tree, func(node *syntax.Node) bool {
		if positions[syntax.NodePosition(node)] {
			tags++
			if syntax.Parent(tree, node) == nil {
				t.Fatal("tag has no parent")
			}
		}
		return true
	})
	if tags != 3 {
		t.Fatalf("walk found %d tags, want 3", tags)
	}
}

func TestCommunicationBodyExcludesCommunication(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample

func read(values <-chan int) int {
	select {
	case result := <-values:
		return result
	default:
		return 0
	}
}
`)
	files := token.NewFileSet()
	file, err := syntax.ParseFile(files, "select.tgo", source, syntax.AllErrors)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	found := false
	syntax.Inspect(file, func(node *syntax.Node) bool {
		statement, ok := syntax.StatementOf(node)
		if !ok || statement.Tag() != syntax.StatementTagCommunication {
			return true
		}
		clause := statement.CommunicationPayload().Value
		if clause.Communication == nil {
			return true
		}
		found = true
		if len(clause.Body) != 1 || clause.Body[0].Tag() != syntax.StatementTagReturn {
			t.Fatalf("communication body = %#v, want one return statement", clause.Body)
		}
		return true
	})
	if !found {
		t.Fatal("communication clause was not found")
	}
}
