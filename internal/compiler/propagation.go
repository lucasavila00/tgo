package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

type propagationFunction struct {
	body       *ast.BlockStmt
	resultAST  []ast.Expr
	resultType *types.Tuple
	names      map[string]bool
}

// lowerPropagations builds and emits one complete typed plan for each function.
func (p *packageUnit) lowerPropagations() {
	for _, source := range p.Sources {
		for _, function := range p.loweringFunctions(source) {
			plan := buildFunctionLoweringPlan(p, source, function)
			newLoweringEmitter(p, source, plan).emit()
		}
		p.reportUnloweredExtensions(source)
	}
}

func functionGotoLabels(body *ast.BlockStmt) map[string]bool {
	labels := make(map[string]bool)
	ast.Inspect(body, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncLit); ok && function.Body != body {
			return false
		}
		branch, ok := node.(*ast.BranchStmt)
		if ok && branch.Tok == token.GOTO && branch.Label != nil {
			labels[branch.Label.Name] = true
		}
		return true
	})
	return labels
}

func (p *packageUnit) loweringFunctions(source *source) []propagationFunction {
	result := []propagationFunction(nil)
	ast.Inspect(source.File, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if _, _, found := comprehensionMarker(source, call); found {
				return false
			}
		}
		switch function := node.(type) {
		case *ast.FuncDecl:
			signature, _ := p.info.TypeOf(function.Name).(*types.Signature)
			result = appendLoweringFunction(result, function.Body, function.Type, signature)
		case *ast.FuncLit:
			signature, _ := p.info.TypeOf(function.Type).(*types.Signature)
			result = appendLoweringFunction(result, function.Body, function.Type, signature)
		}
		return true
	})
	return result
}

func appendLoweringFunction(functions []propagationFunction, body *ast.BlockStmt,
	typeNode *ast.FuncType, signature *types.Signature) []propagationFunction {
	if body == nil || signature == nil {
		return functions
	}
	return append(functions, propagationFunction{
		body: body, resultAST: flattenedResultTypes(typeNode.Results),
		resultType: signature.Results(), names: functionNames(body, signature),
	})
}

func flattenedResultTypes(fields *ast.FieldList) []ast.Expr {
	if fields == nil {
		return nil
	}
	result := []ast.Expr(nil)
	for _, field := range fields.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			result = append(result, field.Type)
		}
	}
	return result
}

func (p *packageUnit) reportUnloweredExtensions(source *source) {
	ast.Inspect(source.File, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if metadata, found := propagationMarker(source, call); found {
			p.failAt(metadata.Bang, "error propagation needs a function body")
		}
		if metadata, _, found := comprehensionMarker(source, call); found {
			p.failAt(metadata.Position, "comprehension needs a function body")
		}
		return true
	})
	ast.Inspect(source.File, func(node ast.Node) bool {
		statement, ok := node.(*ast.ReturnStmt)
		if ok {
			if commas, found := source.FailureReturns[statement]; found {
				p.failAt(commas[0], "failure return needs a valid function signature")
			}
		}
		return true
	})
}

func propagationMarker(source *source, expression ast.Expr) (propagationSource, bool) {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return propagationSource{}, false
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok {
		return propagationSource{}, false
	}
	metadata, ok := source.Propagations[name.Name]
	return metadata, ok
}

func unwrappedCompilerCall(expression ast.Expr) (*ast.CallExpr, bool) {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parenthesized.X
	}
	call, ok := expression.(*ast.CallExpr)
	return call, ok
}

func isPredeclaredError(value types.Type) bool {
	object := types.Universe.Lookup("error")
	return object != nil && types.Identical(value, object.Type())
}

func functionNames(body *ast.BlockStmt, signature *types.Signature) map[string]bool {
	names := make(map[string]bool)
	addTupleNames(names, signature.Params())
	addTupleNames(names, signature.Results())
	if receiver := signature.Recv(); receiver != nil && receiver.Name() != "" {
		names[receiver.Name()] = true
	}
	ast.Inspect(body, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok {
			names[identifier.Name] = true
		}
		return true
	})
	return names
}

func addTupleNames(names map[string]bool, tuple *types.Tuple) {
	for index := range tuple.Len() {
		if name := tuple.At(index).Name(); name != "" {
			names[name] = true
		}
	}
}
