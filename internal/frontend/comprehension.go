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

type comprehensionReplacement struct {
	edit   sourceEdit
	name   string
	record *Comprehension
}

// comprehension plans the innermost comprehension as a marker.
// Every retained source byte keeps its original position.
func (s mappedSource) comprehension(
	original *token.File,
	used map[string]bool,
) (*comprehensionReplacement, error) {
	tokens := sourceTokens(s.text)
	blocks := statementBlockOffsets(s.text)
	for first := len(tokens) - 1; first >= 0; first-- {
		if tokens[first].kind != token.FOR {
			continue
		}
		open := first - 1
		for open >= 0 && tokens[open].depth >= tokens[first].depth {
			open--
		}
		if open < 0 || tokens[open].kind != token.LBRACE || s.input[s.offsets[tokens[open].offset]] != '{' ||
			blocks[tokens[open].offset] {
			continue
		}
		first = firstComprehensionClause(tokens, open, first)
		start := s.comprehensionTypeStart(tokens, open)
		if start < 0 {
			continue
		}
		end := first + 1
		for end < len(tokens) && !(tokens[end].kind == token.RBRACE && tokens[end].depth == tokens[open].depth+1) {
			end++
		}
		if end == len(tokens) {
			return nil, fmt.Errorf(
				"%s: comprehension requires a closing brace",
				original.Position(original.Pos(s.offsets[start])),
			)
		}
		clauses, valid := s.comprehensionClauses(tokens, first, end)
		if !valid {
			continue
		}
		name := freshMarkerName(used, "tgoComprehension")
		output := freshMarkerName(used, "tgoComprehension")
		return &comprehensionReplacement{
			edit:   s.comprehensionEdit(tokens, start, open, first, end, name, output, clauses),
			name:   name,
			record: &Comprehension{Position: original.Position(original.Pos(s.offsets[start])), Output: output},
		}, nil
	}
	return nil, nil
}

// clauseEnd returns the byte end of a clause without its final commas or semicolons.
func clauseEnd(tokens []sourceToken, start, end int) int {
	for end > start && (tokens[end-1].kind == token.SEMICOLON || tokens[end-1].kind == token.COMMA) {
		end--
	}
	if end == start {
		return tokens[start].offset
	}
	return tokens[end-1].end
}

// comprehensionEdit builds a marker call with nested range and filter statements.
// Its position map preserves retained bytes and maps inserted syntax to the input.
func (s mappedSource) comprehensionEdit(
	tokens []sourceToken,
	start, open, first, end int,
	name, output string,
	clauses []int,
) sourceEdit {
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
	comments := func(from, to int) { s.retainComprehensionComments(from, to, retain, insert) }
	insert(name+"(", start)
	retain(start, tokens[open].offset)
	insert(",func(){", tokens[open].offset)
	for i, clause := range clauses {
		stop := end
		if i+1 < len(clauses) {
			stop = clauses[i+1]
		}
		from, to := tokens[clause].offset, clauseEnd(tokens, clause, stop)
		expressionOffset := comprehensionOperandOffset(tokens, clause, stop)
		retain(from, expressionOffset)
		insert(" (", expressionOffset)
		retain(expressionOffset, to)
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
	return sourceEdit{start, tokens[end].offset + 1, text.String(), offsets}
}

// comprehensionRanges collects nested range statements through the prepared filter blocks.
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

// comprehensionClauses finds range and filter clauses at one depth and checks their operands.
// An invalid clause returns false instead of a replacement plan.
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
		if !s.validComprehensionClause(tokens, clause, stop) {
			return nil, false
		}
	}
	return clauses, true
}

// firstComprehensionClause finds the first range clause in the same collection literal.
func firstComprehensionClause(tokens []sourceToken, open, first int) int {
	// An earlier for at this depth is the first clause of the same literal.
	for i := open + 1; i < first; i++ {
		if tokens[i].kind == token.FOR && tokens[i].depth == tokens[first].depth {
			first = i
			break
		}
	}

	return first
}

// comprehensionTypeStart finds the type before a collection brace, or returns -1.
// It uses Go parsing to distinguish a collection type from nearby expressions.
func (s *mappedSource) comprehensionTypeStart(tokens []sourceToken, open int) int {
	start := -1
	for i := open - 1; i >= 0; i-- {
		expr, err := parser.ParseExpr(s.text[tokens[i].offset:tokens[open].offset] + "{}")
		if _, ok := expr.(*ast.CompositeLit); err == nil && ok {
			start = tokens[i].offset
		}
		if tokens[i].depth < tokens[open].depth || tokens[i].kind == token.SEMICOLON || tokens[i].kind == token.ASSIGN ||
			tokens[i].kind == token.DEFINE {
			break
		}
	}

	return start
}

// retainComprehensionComments copies comments from a replaced span through the supplied callbacks.
// It inserts line breaks so line comments do not hide generated syntax.
func (s *mappedSource) retainComprehensionComments(from, to int, retain func(int, int), insert func(string, int)) {

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
			length := len(literal)
			if strings.HasPrefix(s.text[begin:to], "/*") {
				length = strings.Index(s.text[begin:to], "*/") + 2
			}
			retain(begin, begin+length)
			insert("\n", begin)
		}
	}
}

// comprehensionOperandOffset returns the operand start after if or range.
func comprehensionOperandOffset(tokens []sourceToken, clause, stop int) int {
	expressionOffset := tokens[clause].end
	if tokens[clause].kind == token.FOR {
		for j := clause + 1; j < stop; j++ {
			if tokens[j].kind == token.RANGE && tokens[j].depth == tokens[clause].depth {
				expressionOffset = tokens[j].end
				break
			}
		}
	}
	return expressionOffset
}

// validComprehensionClause checks that a range or filter clause has a parseable operand.
func (s *mappedSource) validComprehensionClause(tokens []sourceToken, clause, stop int) bool {
	expressionIndex := clause + 1
	if tokens[clause].kind == token.FOR {
		for expressionIndex < stop && tokens[expressionIndex].kind != token.RANGE {
			expressionIndex++
		}
		if expressionIndex == stop {
			return false
		}
		expressionIndex++
	}
	if expressionIndex >= stop {
		return false
	}
	last := clauseEnd(tokens, expressionIndex, stop)
	if _, err := parser.ParseExpr(s.text[tokens[expressionIndex].offset:last]); err != nil {
		return false
	}
	return true
}

// statementBlockOffsets records statement braces in a partial Go parse.
// Comprehension recognition uses them to exclude ordinary loop bodies.
func statementBlockOffsets(text string) map[int]bool {
	blocks := make(map[int]bool)
	set := token.NewFileSet()
	if file, _ := parser.ParseFile(set, "", text, parser.AllErrors|parser.SkipObjectResolution); file != nil {
		ast.Inspect(file, func(node ast.Node) bool {
			if block, ok := node.(*ast.BlockStmt); ok {
				blocks[set.Position(block.Lbrace).Offset] = true
			}
			return true
		})
	}
	return blocks
}
