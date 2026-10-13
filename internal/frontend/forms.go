package frontend

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/frontend/parser"
)

// nameForms assigns fresh helper names and records the parser's feature nodes for checking.
func (result *Parsed) nameForms(fset *token.FileSet, forms parser.Forms, used map[string]bool) {
	for _, conditional := range forms.Conditionals {
		name := freshMarkerName(used, "tgoConditional")
		conditional.Call.Fun.(*ast.Ident).Name = name
		result.Conditionals[name] = fset.Position(conditional.Position)
	}
	for _, comprehension := range forms.Comprehensions {
		name := freshMarkerName(used, "tgoComprehension")
		output := freshMarkerName(used, "tgoComprehension")
		comprehension.Call.Fun.(*ast.Ident).Name = name
		comprehension.Output.Fun.(*ast.Ident).Name = output
		result.Comprehensions[name] = &Comprehension{
			Position: fset.Position(comprehension.Position), Type: comprehension.Call.Args[0],
			Body: comprehension.Call.Args[1].(*ast.FuncLit).Body, Output: output, Ranges: comprehension.Ranges,
		}
	}
	for _, shortReturn := range forms.ShortReturns {
		name := freshMarkerName(used, "tgoShortReturn")
		shortReturn.Call.Fun.(*ast.Ident).Name = name
		result.ShortReturns[name] = ShortReturn{
			ZeroSuccess: shortReturn.ZeroSuccess, Position: fset.Position(shortReturn.Position),
		}
	}
}

// syntaxContainers records unnamed parser calls that contain added TGo forms.
func syntaxContainers(forms parser.Forms) map[ast.Node]bool {
	nodes := make(map[ast.Node]bool)
	for _, conditional := range forms.Conditionals {
		nodes[conditional.Call] = true
	}
	for _, comprehension := range forms.Comprehensions {
		nodes[comprehension.Call], nodes[comprehension.Output] = true, true
	}
	for _, shortReturn := range forms.ShortReturns {
		nodes[shortReturn.Call] = true
	}
	return nodes
}
