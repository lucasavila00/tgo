package frontend

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
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
	Ranges   []*ast.RangeStmt
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
		clauses, valid := s.comprehensionClauses(tokens, first, end)
		if !valid {
			continue
		}
		name := freshComprehensionName(used)
		output := freshComprehensionName(used)
		records[name] = &Comprehension{Position: original.Position(original.Pos(s.offsets[start])), Output: output}
		s.wrapComprehension(tokens, start, open, first, end, name, output, clauses)
		return true, nil
	}
	return false, nil
}

func clauseEnd(tokens []sourceToken, start, end int) int {
	for end > start && (tokens[end-1].kind == token.SEMICOLON || tokens[end-1].kind == token.COMMA) {
		end--
	}
	if end == start {
		return tokens[start].offset
	}
	return tokens[end-1].end
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

func (s *mappedSource) wrapComprehension(tokens []sourceToken, start, open, first, end int, name, output string, clauses []int) {
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
	comments := func(from, to int) {
		set := token.NewFileSet()
		file := set.AddFile("", -1, to-from)
		var scan scanner.Scanner
		scan.Init(file, []byte(s.text[from:to]), nil, scanner.ScanComments)
		for {
			position, kind, literal := scan.Scan()
			if kind == token.EOF {
				break
			}
			if kind == token.COMMENT {
				begin := from + file.Offset(position)
				retain(begin, begin+len(literal))
				insert("\n", begin)
			}
		}
	}

	insert(name+"(", start)
	retain(start, tokens[open].offset)
	insert(",func(){", tokens[open].offset)
	for i, clause := range clauses {
		stop := end
		if i+1 < len(clauses) {
			stop = clauses[i+1]
		}
		from, to := tokens[clause].offset, clauseEnd(tokens, clause, stop)
		expression := tokens[clause].end
		if tokens[clause].kind == token.FOR {
			for j := clause + 1; j < stop; j++ {
				if tokens[j].kind == token.RANGE && tokens[j].depth == tokens[clause].depth {
					expression = tokens[j].end
					break
				}
			}
		}
		retain(from, expression)
		insert(" (", expression)
		retain(expression, to)
		insert("){", to)
		comments(to, tokens[stop].offset)
	}
	insert(output+"(", tokens[open].offset)
	from, to := tokens[open].offset+1, clauseEnd(tokens, open+1, first)
	previous := from
	for i := open + 1; i < first; i++ {
		if tokens[i].kind == token.COLON && tokens[i].depth == tokens[first].depth {
			retain(previous, tokens[i].offset)
			insert(",", tokens[i].offset)
			previous = tokens[i].offset + 1
		}
	}
	retain(previous, to)
	insert(");", to)
	comments(to, tokens[first].offset)
	insert(strings.Repeat("}", len(clauses))+"})", tokens[end].offset)
	s.replace(start, tokens[end].offset+1, text.String(), offsets)
}

func comprehensionRanges(body *ast.BlockStmt) []*ast.RangeStmt {
	var ranges []*ast.RangeStmt
	for len(body.List) == 1 {
		switch clause := body.List[0].(type) {
		case *ast.RangeStmt:
			ranges = append(ranges, clause)
			body = clause.Body
		case *ast.IfStmt:
			body = clause.Body
		default:
			return ranges
		}
	}
	return ranges
}

func (s *mappedSource) comprehensionClauses(tokens []sourceToken, first, end int) ([]int, bool) {
	var clauses []int
	for i := first; i < end; i++ {
		if tokens[i].depth == tokens[first].depth && (tokens[i].kind == token.FOR || tokens[i].kind == token.IF) {
			clauses = append(clauses, i)
		}
	}
	for i, clause := range clauses {
		stop := end
		if i+1 < len(clauses) {
			stop = clauses[i+1]
		}
		expression := clause + 1
		if tokens[clause].kind == token.FOR {
			for expression < stop && tokens[expression].kind != token.RANGE {
				expression++
			}
			if expression == stop {
				return nil, false
			}
			expression++
		}
		if expression >= stop {
			return nil, false
		}
		last := clauseEnd(tokens, expression, stop)
		if _, err := parser.ParseExpr(s.text[tokens[expression].offset:last]); err != nil {
			return nil, false
		}
	}
	return clauses, true
}
