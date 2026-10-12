package frontend

import (
	"fmt"
	"go/ast"
	"go/scanner"
	"go/token"
)

func conditionalStart(file *ast.File, temporary *token.FileSet, source string) int {
	candidates := expressionConditionalTokens(source)
	start := -1
	// Prepare inner operands first. Parser recovery can truncate an outer
	// conditional when a nested value remains in its branch.
	for offset, operand := range candidates {
		if operand && offset > start {
			start = offset
		}
	}
	if start >= 0 {
		return start
	}
	return recoveredConditionalStart(file, temporary, source, candidates)
}

func recoveredConditionalStart(file *ast.File, temporary *token.FileSet, source string, candidates map[int]bool) int {
	start := -1
	var parents []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			parents = parents[:len(parents)-1]
			return false
		}
		if bad, ok := node.(*ast.BadExpr); ok {
			offset := temporary.Position(bad.Pos()).Offset
			syntaxOperand := false
			switch parents[len(parents)-1].(type) {
			case *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.CompositeLit, *ast.KeyValueExpr, *ast.SliceExpr:
				syntaxOperand = true
			}
			startsConditional := false
			if offset+2 <= len(source) {
				startsConditional = source[offset:offset+2] == "if"
			}
			if offset+6 <= len(source) {
				startsConditional = startsConditional || source[offset:offset+6] == "switch"
			}
			canStartOperand := candidates[offset] || syntaxOperand
			if startsConditional && canStartOperand && offset > start {
				start = offset
			}
		}
		parents = append(parents, node)
		return true
	})
	return start
}

func (s *mappedSource) wrapConditional(filename string, original *token.File, start int, used map[string]bool,
	conditionals map[string]token.Position) error {
	end := conditionalEnd(filename, s.text, start, sourceTokens(s.text))
	if end < 0 {
		return fmt.Errorf("%s: invalid conditional expression", original.Position(original.Pos(s.offsets[start])))
	}
	name := ""
	for n := 1; ; n++ {
		name = fmt.Sprintf("tgoConditional%d", n)
		if !used[name] {
			used[name] = true
			break
		}
	}
	conditionals[name] = original.Position(original.Pos(s.offsets[start]))
	before, after := name+"(func(){", "})"
	offsets := insertedOffsets(len(before), s.offsets[start])
	offsets = append(offsets, s.offsets[start:end]...)
	offsets = append(offsets, insertedOffsets(len(after), s.offsets[end-1])...)
	s.replace(start, end, before+s.text[start:end]+after, offsets)
	return nil
}

// expressionConditionalTokens distinguishes operand tokens from conditional statements.
// Parser recovery after a bad operand can report later statements as operands.
func expressionConditionalTokens(source string) map[int]bool {
	set := token.NewFileSet()
	file := set.AddFile("", -1, len(source))
	var scan scanner.Scanner
	scan.Init(file, []byte(source), nil, 0)
	result := make(map[int]bool)
	previous := token.ILLEGAL
	for {
		position, current, _ := scan.Scan()
		if current == token.EOF {
			return result
		}
		if current == token.IF || current == token.SWITCH {
			switch previous {
			case token.ASSIGN, token.DEFINE, token.COMMA,
				token.LPAREN, token.LBRACK, token.RETURN, token.RANGE,
				token.IF, token.FOR, token.SWITCH, token.CASE, token.GO, token.DEFER:
				result[file.Offset(position)] = true
			default:
				result[file.Offset(position)] = previous.Precedence() > 0 || previous == token.NOT ||
					previous == token.ARROW || (previous >= token.ADD_ASSIGN &&
					previous <= token.AND_NOT_ASSIGN)
			}
		}
		previous = current
	}
}
