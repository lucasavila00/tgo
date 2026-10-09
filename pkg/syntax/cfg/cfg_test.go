package cfg_test

import (
	"go/token"
	"testing"

	"tgo/pkg/syntax"
	"tgo/pkg/syntax/cfg"
)

func TestNewMarksReturnSuccessorUnreachable(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample

func choose(value bool) int {
	if value { return 1 }
	return 2
}
`)
	file, err := syntax.ParseGoFile(
		token.NewFileSet(), "choose.go", source, syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("ParseGoFile: %v", err)
	}
	function := syntax.FunctionDeclarationValueOf(file.Declarations[0])
	if function == nil || function.Body == nil {
		t.Fatal("function body is absent")
	}
	graph := cfg.New(function.Body, func(*syntax.Expression) bool { return true })
	liveReturns := 0
	for _, block := range graph.Blocks {
		if !block.Live { continue }
		for _, node := range block.Nodes {
			statement, ok := syntax.StatementOf(&node)
			if ok && syntax.ReturnStatementOf(statement) != nil { liveReturns++ }
		}
	}
	if liveReturns != 2 {
		t.Fatalf("live returns = %d, want 2", liveReturns)
	}
}
