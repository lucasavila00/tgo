package compiler

import (
	"go/ast"
	"go/token"

	"tgo/pkg/syntax"
)

// lowerSuccessReturnCommas projects each trailing comma as one semicolon.
func lowerSuccessReturnCommas(
	files *token.FileSet,
	file *token.File,
	tree *syntax.File,
	edits []edit,
) ([]edit, map[[2]int]bool) {
	locations := make(map[[2]int]bool)
	syntax.Inspect(tree, func(node *syntax.Node) bool {
		statement, ok := syntax.StatementOf(node)
		if !ok {
			return true
		}
		returned := syntax.ReturnStatementOf(statement)
		if returned == nil || !returned.SuccessComma.IsValid() {
			return true
		}
		offset := file.Offset(returned.SuccessComma)
		edits = append(edits, edit{start: offset, end: offset + 1, text: ";"})
		position := files.Position(returned.Return)
		locations[[2]int{position.Line, position.Column}] = true
		return true
	})
	return edits, locations
}

// lowerFailureReturnCommas removes each leading return comma before Go parsing.
func lowerFailureReturnCommas(
	files *token.FileSet,
	file *token.File,
	tree *syntax.File,
	edits []edit,
) ([]edit, map[[2]int][]token.Pos) {
	locations := make(map[[2]int][]token.Pos)
	syntax.Inspect(tree, func(node *syntax.Node) bool {
		statement, ok := syntax.StatementOf(node)
		if !ok {
			return true
		}
		returned := syntax.ReturnStatementOf(statement)
		if returned == nil || len(returned.FailureCommas) == 0 {
			return true
		}
		for _, comma := range returned.FailureCommas {
			offset := file.Offset(comma)
			edits = append(edits, edit{start: offset, end: offset + 1, text: ""})
		}
		position := files.Position(returned.Return)
		locations[[2]int{position.Line, position.Column}] = returned.FailureCommas
		return true
	})
	return edits, locations
}

func projectedSuccessReturns(
	files *token.FileSet,
	file *ast.File,
	locations map[[2]int]bool,
) []*ast.ReturnStmt {
	result := make([]*ast.ReturnStmt, 0, len(locations))
	ast.Inspect(file, func(node ast.Node) bool {
		statement, ok := node.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		position := files.Position(statement.Return)
		if locations[[2]int{position.Line, position.Column}] {
			result = append(result, statement)
		}
		return true
	})
	return result
}

func projectedFailureReturns(
	files *token.FileSet,
	file *ast.File,
	locations map[[2]int][]token.Pos,
) map[*ast.ReturnStmt][]token.Pos {
	result := make(map[*ast.ReturnStmt][]token.Pos, len(locations))
	ast.Inspect(file, func(node ast.Node) bool {
		statement, ok := node.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		position := files.Position(statement.Return)
		if commas := locations[[2]int{position.Line, position.Column}]; len(commas) > 0 {
			result[statement] = commas
		}
		return true
	})
	return result
}

// lowerSuccessReturns appends the predeclared nil before the first type check.
func (p *packageUnit) lowerSuccessReturns(source *source) {
	for _, statement := range source.SuccessReturns {
		nilValue := p.generatedUniverse("nil", statement.Return)
		nilValue.NamePos = statement.Return
		statement.Results = append(
			statement.Results,
			nilValue,
		)
	}
}
