package frontend

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/scanner"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// Marker records a postfix operation in Go-compatible source.
type Marker struct {
	Name     string
	Wrap     bool
	CallName string
	Position token.Position
	// EndOffset is the byte offset just after the operator in the input.
	EndOffset int
}

type postfixOperator struct {
	start, end int
	wrap       bool
}

type postfixReplacement struct {
	edit   sourceEdit
	record Marker
}

// postfixReplacements records operators in scan order and plans their comments.
func postfixReplacements(filename string, input []byte, used map[string]bool) []postfixReplacement {
	fset := token.NewFileSet()
	file := fset.AddFile(filename, -1, len(input))
	var scan scanner.Scanner
	scan.Init(file, input, nil, scanner.ScanComments)
	var replacements []postfixReplacement
	previous := token.ILLEGAL
	for {
		pos, tok, _ := scan.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT {
			continue
		}
		offset := file.Offset(pos)
		if tok == token.NOT && previous == token.RPAREN {
			operator := readPostfixOperator(&scan, input, offset)
			name := freshMarkerName(used, "tgoPropagation")
			text := "/*" + name + "*/"
			replacements = append(replacements, postfixReplacement{
				edit: sourceEdit{operator.start, operator.end, text, insertedOffsets(len(text), operator.start)},
				record: Marker{Name: name, Wrap: operator.wrap, Position: file.Position(pos),
					EndOffset: operator.end},
			})
			previous = token.ILLEGAL
			continue
		}
		previous = tok
	}
	return replacements
}

// readPostfixOperator records the operator's meaning and input span after its first !.
// It consumes the second ! only when the two bytes are adjacent.
func readPostfixOperator(scan *scanner.Scanner, input []byte, start int) postfixOperator {
	operator := postfixOperator{start: start, end: start + 1}
	if operator.end < len(input) && input[operator.end] == '!' {
		_, next, _ := scan.Scan()
		if next != token.NOT {
			panic("invalid propagation suffix")
		}
		operator.end++
		operator.wrap = true
	}
	return operator
}

// markerExpressions matches recorded comments to calls in the parsed file.
// It records the source call name and panics if an operator does not follow a call.
func markerExpressions(fset *token.FileSet, file *ast.File, operations map[string]Marker) map[ast.Expr]Marker {
	result := make(map[ast.Expr]Marker)
	for _, group := range file.Comments {
		for _, comment := range group.List {
			name := strings.TrimSuffix(strings.TrimPrefix(comment.Text, "/*"), "*/")
			operation, ok := operations[name]
			if !ok {
				continue
			}
			selected := precedingCall(file, comment.Pos())
			raw := ast.Unparen(selected)
			call, ok := raw.(*ast.CallExpr)
			if !ok {
				panic(fmt.Sprintf("%s: propagation requires a call", fset.Position(comment.Pos())))
			}
			var callName bytes.Buffer
			if err := format.Node(&callName, fset, call.Fun); err != nil {
				panic(err)
			}
			if operation.CallName == "" {
				operation.CallName = callName.String()
			}
			result[selected] = operation
		}
	}
	return result
}

// wrapMarker replaces the selected expression with a marker call for source checking.
// It changes file in place and removes the comment for this operation.
func wrapMarker(file *ast.File, original *token.File, expression ast.Expr, operation Marker) {
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		if cursor.Node() == expression {
			cursor.Replace(&ast.CallExpr{
				Fun:    &ast.Ident{Name: operation.Name, NamePos: expression.Pos()},
				Lparen: expression.Pos(), Args: []ast.Expr{ast.Unparen(expression)},
				Rparen: original.Pos(operation.EndOffset - 1),
			})
			return false
		}
		return true
	}, nil)
	comments := file.Comments[:0]
	for _, group := range file.Comments {
		kept := group.List[:0]
		for _, comment := range group.List {
			if comment.Text != "/*"+operation.Name+"*/" {
				kept = append(kept, comment)
			}
		}
		group.List = kept
		if len(kept) != 0 {
			comments = append(comments, group)
		}
	}
	file.Comments = comments
}

// precedingCall finds the call or parenthesized operand ending nearest to a postfix marker.
// For equal ends, it selects the expression with the earliest start.
func precedingCall(file *ast.File, position token.Pos) ast.Expr {
	var selected ast.Expr
	ast.Inspect(file, func(node ast.Node) bool {
		expr, ok := node.(ast.Expr)
		switch expr.(type) {
		case *ast.CallExpr, *ast.ParenExpr:
		default:
			return true
		}
		if !ok || expr.End() > position {
			return true
		}
		if selected == nil || expr.End() > selected.End() || (expr.End() == selected.End() && expr.Pos() < selected.Pos()) {
			selected = expr
		}
		return true
	})
	return selected
}
