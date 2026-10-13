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
			count := 1
			if offset+1 < len(input) && input[offset+1] == '!' {
				count = 2
			}
			name := freshMarkerName(used, "tgoPropagation")
			text := "/*" + name + "*/"
			replacements = append(replacements, postfixReplacement{
				edit:   sourceEdit{offset, offset + count, text, insertedOffsets(len(text), offset)},
				record: Marker{Name: name, Wrap: count == 2, Position: file.Position(pos)},
			})
			if count == 2 {
				_, next, _ := scan.Scan()
				if next != token.NOT {
					panic("invalid propagation suffix")
				}
			}
			previous = token.ILLEGAL
			continue
		}
		previous = tok
	}
	return replacements
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
func wrapMarker(file *ast.File, expression ast.Expr, operation Marker) {
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		if cursor.Node() == expression {
			width := token.Pos(1)
			if operation.Wrap {
				width++
			}
			cursor.Replace(&ast.CallExpr{
				Fun:    &ast.Ident{Name: operation.Name, NamePos: expression.Pos()},
				Lparen: expression.Pos(), Args: []ast.Expr{ast.Unparen(expression)},
				Rparen: expression.End() + width - 1,
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
