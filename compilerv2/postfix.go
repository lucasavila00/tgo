package compilerv2

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type propagation struct {
	name     string
	wrap     bool
	callName string
}

// markPostfix changes suffixes into comments that survive Go parsing.
func markPostfix(filename string, input []byte, used map[string]bool, operations map[string]propagation) []byte {
	fset := token.NewFileSet()
	file := fset.AddFile(filename, -1, len(input))
	var scan scanner.Scanner
	scan.Init(file, input, nil, scanner.ScanComments)
	var output bytes.Buffer
	previous := token.ILLEGAL
	copied := 0
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
			name := ""
			for n := 1; ; n++ {
				name = fmt.Sprintf("tgoPropagation%d", n)
				if !used[name] {
					used[name] = true
					break
				}
			}
			operations[name] = propagation{name: name, wrap: count == 1}
			output.Write(input[copied:offset])
			fmt.Fprintf(&output, "/*%s*/", name)
			copied = offset + count
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
	output.Write(input[copied:])
	return output.Bytes()
}

func markerExpressions(fset *token.FileSet, file *ast.File, operations map[string]propagation) map[ast.Expr]propagation {
	result := make(map[ast.Expr]propagation)
	for _, group := range file.Comments {
		for _, comment := range group.List {
			name := strings.TrimSuffix(strings.TrimPrefix(comment.Text, "/*"), "*/")
			operation, ok := operations[name]
			if !ok {
				continue
			}
			var selected ast.Expr
			ast.Inspect(file, func(node ast.Node) bool {
				expr, ok := node.(ast.Expr)
				switch expr.(type) {
				case *ast.CallExpr, *ast.ParenExpr:
				default:
					return true
				}
				if !ok || expr.End() > comment.Pos() {
					return true
				}
				if selected == nil || expr.End() > selected.End() || (expr.End() == selected.End() && expr.Pos() < selected.Pos()) {
					selected = expr
				}
				return true
			})
			raw := unparen(selected)
			call, ok := raw.(*ast.CallExpr)
			if !ok {
				panic(fmt.Sprintf("%s: propagation requires a call", fset.Position(comment.Pos())))
			}
			var callName bytes.Buffer
			if err := format.Node(&callName, fset, call.Fun); err != nil {
				panic(err)
			}
			if operation.callName == "" {
				operation.callName = callName.String()
			}
			result[selected] = operation
		}
	}
	return result
}

func unparen(expr ast.Expr) ast.Expr {
	for {
		p, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = p.X
	}
}

func wrapMarker(file *ast.File, expression ast.Expr, operation propagation) {
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		if cursor.Node() == expression {
			cursor.Replace(&ast.CallExpr{Fun: ast.NewIdent(operation.name), Args: []ast.Expr{unparen(expression)}})
			return false
		}
		return true
	}, nil)
	comments := file.Comments[:0]
	for _, group := range file.Comments {
		remove := false
		for _, comment := range group.List {
			if comment.Text == "/*"+operation.name+"*/" {
				remove = true
			}
		}
		if !remove {
			comments = append(comments, group)
		}
	}
	file.Comments = comments
}

func markerDeclaration(operation propagation, count int) ast.Decl {
	var text strings.Builder
	fmt.Fprintf(&text, "package marker\nfunc %s", operation.name)
	if count >= 0 {
		text.WriteString("[")
		for i := 0; i <= count; i++ {
			if i > 0 {
				text.WriteString(",")
			}
			fmt.Fprintf(&text, "T%d interface{}", i)
		}
		text.WriteString("]")
	}
	text.WriteString("(")
	for i := 0; i < count; i++ {
		fmt.Fprintf(&text, "v%d T%d,", i, i)
	}
	fmt.Fprintf(&text, "err T%d)", count)
	if count > 0 {
		text.WriteString("(")
		for i := 0; i < count; i++ {
			if i > 0 {
				text.WriteString(",")
			}
			fmt.Fprintf(&text, "T%d", i)
		}
		text.WriteString(")")
	}
	text.WriteString("{return ")
	for i := 0; i < count; i++ {
		if i > 0 {
			text.WriteString(",")
		}
		fmt.Fprintf(&text, "v%d", i)
	}
	text.WriteString("}")
	file, err := parser.ParseFile(token.NewFileSet(), "marker.go", text.String(), 0)
	if err != nil {
		panic(err)
	}
	return file.Decls[0]
}
