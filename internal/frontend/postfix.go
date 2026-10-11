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
}

// MarkPostfix changes suffixes into comments that survive Go parsing.
func MarkPostfix(filename string, input []byte, used map[string]bool, operations map[string]Marker) []byte {
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
			operations[name] = Marker{Name: name, Wrap: count == 1}
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

// MarkerExpressions pairs each suffix comment with its call.
func MarkerExpressions(fset *token.FileSet, file *ast.File, operations map[string]Marker) map[ast.Expr]Marker {
	result := make(map[ast.Expr]Marker)
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

// WrapMarker replaces a postfix call with a call to its type-checking marker.
func WrapMarker(file *ast.File, expression ast.Expr, operation Marker) {
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		if cursor.Node() == expression {
			cursor.Replace(&ast.CallExpr{Fun: ast.NewIdent(operation.Name), Args: []ast.Expr{ast.Unparen(expression)}})
			return false
		}
		return true
	}, nil)
	comments := file.Comments[:0]
	for _, group := range file.Comments {
		remove := false
		for _, comment := range group.List {
			if comment.Text == "/*"+operation.Name+"*/" {
				remove = true
			}
		}
		if !remove {
			comments = append(comments, group)
		}
	}
	file.Comments = comments
}

// MarkerDeclaration gives the marker its success result types.
func MarkerDeclaration(operation Marker, count int) ast.Decl {
	parameters := &ast.FieldList{}
	arguments := &ast.FieldList{}
	results := &ast.FieldList{}
	values := []ast.Expr{}
	for i := 0; i <= count; i++ {
		typ := ast.NewIdent(fmt.Sprintf("T%d", i))
		parameters.List = append(parameters.List, &ast.Field{Names: []*ast.Ident{typ}, Type: &ast.InterfaceType{Methods: &ast.FieldList{}}})
		name := ast.NewIdent("err")
		if i < count {
			name = ast.NewIdent(fmt.Sprintf("v%d", i))
			results.List = append(results.List, &ast.Field{Type: typ})
			values = append(values, name)
		}
		arguments.List = append(arguments.List, &ast.Field{Names: []*ast.Ident{name}, Type: typ})
	}
	signature := &ast.FuncType{TypeParams: parameters, Params: arguments}
	if count > 0 {
		signature.Results = results
	}
	return &ast.FuncDecl{Name: ast.NewIdent(operation.Name), Type: signature, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: values}}}}
}
