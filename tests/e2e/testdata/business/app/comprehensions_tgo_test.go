package app

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

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
	generated, err := os.ReadFile("comprehensions_tgo.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, forbidden := range []string{"__tgo_", "func() comprehension"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("generated projection remains: %s", forbidden)
		}
	}
	if !strings.Contains(text, "for _, account := range accounts") ||
		!strings.Contains(text, "for _, sale := range account.Sales") {
		t.Fatal("generated output does not contain the fused loop nest")
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), "comprehensions_tgo.go", generated, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !generatedFunctionCalls(parsed, "ComprehensionCopy", "copy") {
		t.Fatal("generated ComprehensionCopy does not call copy")
	}
	if !strings.Contains(text, "] = comprehensionRecord(") {
		t.Fatal("generated transformed comprehension does not use indexed evaluation")
	}
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

func generatedFunctionCalls(file *ast.File, function string, called string) bool {
	for _, declaration := range file.Decls {
		candidate, ok := declaration.(*ast.FuncDecl)
		if !ok || candidate.Name.Name != function {
			continue
		}
		found := false
		ast.Inspect(candidate.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if ok && identifier.Name == called {
				found = true
			}
			return !found
		})
		return found
	}
	return false
}
