package app

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func generatedFunction(t *testing.T, path, name string) *ast.FuncDecl {
	t.Helper()
	file := generatedFile(t, path)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	t.Fatalf("generated function %s is absent", name)
	return nil
}

func generatedFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func rejectGeneratedPrefix(t *testing.T, file *ast.File) {
	t.Helper()
	ast.Inspect(file, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && strings.HasPrefix(identifier.Name, "__tgo_") {
			t.Fatalf("generated file contains %s", identifier.Name)
		}
		return true
	})
}

func rejectGeneratedWrappers(t *testing.T, function *ast.FuncDecl) {
	t.Helper()
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.FuncLit:
			t.Fatalf("generated function %s contains a wrapper", function.Name.Name)
		case *ast.Ident:
			if strings.HasPrefix(value.Name, "__tgo_") {
				t.Fatalf("generated function %s contains %s", function.Name.Name, value.Name)
			}
		}
		return true
	})
}

func assertIndexedComprehension(t *testing.T, function *ast.FuncDecl) {
	t.Helper()
	found := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		rangeStatement, ok := node.(*ast.RangeStmt)
		key, named := rangeStatementKey(rangeStatement, ok)
		if !named {
			return true
		}
		var captured string
		for _, statement := range rangeStatement.Body.List {
			assignment, ok := statement.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
				continue
			}
			if expressionCallsIdentifier(assignment.Rhs[0], "comprehensionRecord") {
				identifier, named := assignment.Lhs[0].(*ast.Ident)
				if named {
					captured = identifier.Name
				}
				continue
			}
			indexed, indexedOK := assignment.Lhs[0].(*ast.IndexExpr)
			if !indexedOK {
				continue
			}
			index, indexOK := indexed.Index.(*ast.Ident)
			found = captured != "" && indexOK && index.Name == key.Name &&
				expressionUsesIdentifier(assignment.Rhs[0], captured)
			if found {
				break
			}
		}
		return !found
	})
	if !found {
		t.Fatalf("generated function %s lacks ordered indexed evaluation", function.Name.Name)
	}
}

func expressionCallsIdentifier(node ast.Node, name string) bool {
	found := false
	ast.Inspect(node, func(child ast.Node) bool {
		call, ok := child.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, named := call.Fun.(*ast.Ident)
		found = found || named && identifier.Name == name
		return !found
	})
	return found
}

func assertNestedRanges(t *testing.T, function *ast.FuncDecl) {
	t.Helper()
	found := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		outer, ok := node.(*ast.RangeStmt)
		if !ok {
			return true
		}
		ast.Inspect(outer.Body, func(child ast.Node) bool {
			if child != outer.Body {
				if _, nested := child.(*ast.RangeStmt); nested {
					found = true
				}
			}
			return !found
		})
		return !found
	})
	if !found {
		t.Fatalf("generated function %s lacks a fused range nest", function.Name.Name)
	}
}

func rangeStatementKey(statement *ast.RangeStmt, ok bool) (*ast.Ident, bool) {
	if !ok || statement == nil {
		return nil, false
	}
	key, named := statement.Key.(*ast.Ident)
	return key, named && key.Name != "_"
}

func comprehensionFixture() []comprehensionAccount {
	return []comprehensionAccount{
		{ID: "one", Active: true, Sales: []int{1, 2}},
		{ID: "two", Active: false, Sales: []int{3}},
		{ID: "one", Active: true, Sales: []int{4}},
	}
}

func TestComprehensions(t *testing.T) {
	accounts := comprehensionFixture()
	names := ComprehensionNames(accounts)
	if strings.Join(names, ",") != "one,one" {
		t.Fatal(names)
	}
	pairs := ComprehensionPairs(accounts)
	if len(pairs) != 4 || pairs[0] != [2]int{3, 1} || pairs[3] != [2]int{3, 4} {
		t.Fatal(pairs)
	}
	byID := ComprehensionByID(accounts)
	if len(byID) != 2 || byID["one"].Sales[0] != 4 {
		t.Fatal(byID)
	}
	if empty := ComprehensionEmpty(); empty == nil || len(empty) != 0 {
		t.Fatalf("empty result: %#v", empty)
	}
	indexes := ComprehensionIndexes(3)
	if len(indexes) != 3 || indexes[0] != 0 || indexes[2] != 2 {
		t.Fatal(indexes)
	}
	called := ComprehensionMapCall([]string{"one"})
	if called["one"] != "value:one" {
		t.Fatal(called)
	}
	events := []string{}
	values := ComprehensionOrder(&events)
	if strings.Join(values, ",") != "value1,value2" ||
		strings.Join(events, ",") != "source,value,value" {
		t.Fatalf("values=%v events=%v", values, events)
	}
}

func TestComprehensionPropagation(t *testing.T) {
	tests := []struct {
		name string
		run  func(*[]string) error
		want string
	}{
		{name: "source", run: func(events *[]string) error {
			_, err := ComprehensionSourceError(events, true)
			return err
		}, want: "comprehensionLoad: comprehension failure"},
		{name: "filter", run: func(events *[]string) error {
			_, err := ComprehensionFilterError(events, true)
			return err
		}, want: "comprehensionLoad: comprehension failure"},
		{name: "result", run: func(events *[]string) error {
			_, err := ComprehensionResultError(events, true)
			return err
		}, want: "comprehensionLoad: comprehension failure"},
		{name: "map key", run: func(events *[]string) error {
			_, err := ComprehensionMapError(events, true, false)
			return err
		}, want: "comprehensionLoad: comprehension failure"},
		{name: "map value", run: func(events *[]string) error {
			_, err := ComprehensionMapError(events, false, true)
			return err
		}, want: "comprehensionLoad: comprehension failure"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []string{}
			err := test.run(&events)
			if !errors.Is(err, errComprehension) || err.Error() != test.want {
				t.Fatalf("error=%v events=%v", err, events)
			}
		})
	}
}

func TestComprehensionGeneratedLoops(t *testing.T) {
	if value := ComprehensionForInitializer([]int{7, 8}); value != 7 {
		t.Fatalf("for initializer value=%d", value)
	}
	pairsFunction := generatedFunction(t, "comprehensions_tgo.go", "ComprehensionPairs")
	orderFunction := generatedFunction(t, "comprehensions_tgo.go", "ComprehensionOrder")
	rejectGeneratedPrefix(t, generatedFile(t, "comprehensions_tgo.go"))
	rejectGeneratedWrappers(t, pairsFunction)
	rejectGeneratedWrappers(t, orderFunction)
	assertNestedRanges(t, pairsFunction)
	assertIndexedComprehension(t, orderFunction)
	values := []int{1, 2, 3, 4}
	var copied []int
	allocations := testing.AllocsPerRun(1000, func() {
		copied = ComprehensionCopy(values)
	})
	if allocations != 1 || len(copied) != len(values) || copied[0] != values[0] ||
		&copied[0] == &values[0] {
		t.Fatalf("copy cost: %f allocations, %v", allocations, copied)
	}
}
