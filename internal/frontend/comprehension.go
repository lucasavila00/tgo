package frontend

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Comprehension records a typed collection and its range and filter statements.
// Output identifies the leaf call that supplies an element or a key and value.
type Comprehension struct {
	Position token.Position
	Type     ast.Expr
	Body     *ast.BlockStmt
	Output   string
}

// prepareComprehension replaces the innermost comprehension with a marker.
// Every retained source byte keeps its original position.
func (s *mappedSource) prepareComprehension(original *token.File, used map[string]bool, records map[string]*Comprehension) (bool, error) {
	tokens := sourceTokens(s.text)
	blocks := make(map[int]bool)
	set := token.NewFileSet()
	if file, _ := parser.ParseFile(set, "", s.text, parser.AllErrors|parser.SkipObjectResolution); file != nil {
		ast.Inspect(file, func(node ast.Node) bool {
			if block, ok := node.(*ast.BlockStmt); ok {
				blocks[set.Position(block.Lbrace).Offset] = true
			}
			return true
		})
	}
	for first := len(tokens) - 1; first >= 0; first-- {
		if tokens[first].kind != token.FOR {
			continue
		}
		open := first - 1
		for open >= 0 && tokens[open].depth >= tokens[first].depth {
			open--
		}
		if open < 0 || tokens[open].kind != token.LBRACE || s.input[s.offsets[tokens[open].offset]] != '{' || blocks[tokens[open].offset] {
			continue
		}
		// An earlier for at this depth is the first clause of the same literal.
		for i := open + 1; i < first; i++ {
			if tokens[i].kind == token.FOR && tokens[i].depth == tokens[first].depth {
				first = i
				break
			}
		}
		start := -1
		for i := open - 1; i >= 0; i-- {
			expr, err := parser.ParseExpr(s.text[tokens[i].offset:tokens[open].offset] + "{}")
			if _, ok := expr.(*ast.CompositeLit); err == nil && ok {
				start = tokens[i].offset
			}
			if tokens[i].depth < tokens[open].depth || tokens[i].kind == token.SEMICOLON || tokens[i].kind == token.ASSIGN || tokens[i].kind == token.DEFINE {
				break
			}
		}
		if start < 0 {
			continue
		}
		end := first + 1
		for end < len(tokens) && !(tokens[end].kind == token.RBRACE && tokens[end].depth == tokens[open].depth+1) {
			end++
		}
		if end == len(tokens) {
			return false, fmt.Errorf("%s: comprehension requires a closing brace", original.Position(original.Pos(s.offsets[start])))
		}
		name := freshComprehensionName(used)
		output := freshComprehensionName(used)
		records[name] = &Comprehension{Position: original.Position(original.Pos(s.offsets[start])), Output: output}
		var text strings.Builder
		var offsets []int
		insert := func(value string, at int) {
			text.WriteString(value)
			offsets = append(offsets, insertedOffsets(len(value), s.offsets[at])...)
		}
		retain := func(from, to int) {
			text.WriteString(s.text[from:to])
			offsets = append(offsets, s.offsets[from:to]...)
		}
		insert(name+"(", start)
		retain(start, tokens[open].offset)
		insert(",func(){", tokens[open].offset)
		var clauses []int
		for i := first; i < end; i++ {
			if tokens[i].depth == tokens[first].depth && (tokens[i].kind == token.FOR || tokens[i].kind == token.IF) {
				clauses = append(clauses, i)
			}
		}
		for i, clause := range clauses {
			stop := tokens[end].offset
			if i+1 < len(clauses) {
				stop = tokens[clauses[i+1]].offset
			}
			from, to := trimClause(s.text, tokens[clause].offset, stop)
			retain(from, to)
			insert("{", to)
		}
		insert(output+"(", tokens[open].offset)
		from, to := trimClause(s.text, tokens[open].offset+1, tokens[first].offset)
		previous := from
		for i := open + 1; i < first; i++ {
			if tokens[i].kind == token.COLON && tokens[i].depth == tokens[first].depth {
				retain(previous, tokens[i].offset)
				insert(",", tokens[i].offset)
				previous = tokens[i].offset + 1
			}
		}
		retain(previous, to)
		insert(");"+strings.Repeat("}", len(clauses))+"})", tokens[end].offset)
		s.replace(start, tokens[end].offset+1, text.String(), offsets)
		return true, nil
	}
	return false, nil
}

func trimClause(source string, start, end int) (int, int) {
	for start < end && strings.ContainsRune(" \t\r\n", rune(source[start])) {
		start++
	}
	for end > start && strings.ContainsRune(" \t\r\n;,", rune(source[end-1])) {
		end--
	}
	return start, end
}

func freshComprehensionName(used map[string]bool) string {
	for n := 1; ; n++ {
		name := fmt.Sprintf("tgoComprehension%d", n)
		if !used[name] {
			used[name] = true
			return name
		}
	}
}
