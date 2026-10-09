package syntax_test

import (
	"go/token"
	"reflect"
	"testing"

	"tgo/pkg/syntax"
)

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
type Port int where value > 0

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
	match Result.OK{Value: local} {
	case OK(ok):
		local = ok.Value
	case Error(problem):
		local = len(problem.Message)
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
		"Match",
	})
	requireKinds(t, declarations, []string{
		"General", "Function", "Enum", "Struct", "Checked",
	})
	requireKinds(t, specifications, []string{"Import", "Value", "Type"})
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
			want: "bad.tgo:p.declaration: p.closeToken: 3:25: unmatched ]",
		},
		{
			name: "two results",
			source: "package sample\n" +
				"type Result enum {\n" +
				"\tBad int\n" +
				"}\n",
			want: "bad.tgo:p.declaration: p.variant: 3:2: variant needs Name struct { fields }",
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
