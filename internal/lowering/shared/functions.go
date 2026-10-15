package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

type returnFrame struct {
	function ast.Node
	typ      *ast.FuncType
	prepared bool
	prefix   []ast.Stmt
	values   []ast.Expr
}

// functionBody lowers a function with fresh return and jump state and shared file bindings.
// It adds generated zero-result storage and type aliases to the copied body.
func (r *Context) functionBody(node ast.Node, typ *ast.FuncType, body *ast.BlockStmt) *ast.BlockStmt {
	functionRewrite := &Context{
		input:         r.input,
		dispatch:      r.dispatch,
		request:       r.request,
		file:          r.file,
		functionState: &functionState{frame: &returnFrame{function: node, typ: typ}, hasGoto: r.functionHasGoto(body)},
	}
	output := functionRewrite.Block(body)
	if functionRewrite.functionState.frame.prepared {
		output.List = append(functionRewrite.functionState.frame.prefix, output.List...)
	}
	if functionRewrite.input.Inventory.Checked.TypesInfo.TGoReturnBlocks[body] {
		output = functionRewrite.returnBlock(typ, output)
	}
	if aliases := functionRewrite.file.aliasStatements[node]; len(aliases) != 0 {
		output.List = append(aliases, output.List...)
	}
	return output
}

// prepareReturns saves zero values for the results before the error.
func (r *Context) prepareReturns(original *ast.FuncType) ([]ast.Stmt, []ast.Expr) {
	if original.Results == nil {
		panic("propagation requires an error return")
	}
	var prefix []ast.Stmt
	var values []ast.Expr
	for i, field := range original.Results.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for j := 0; j < count; j++ {
			if i == len(original.Results.List)-1 && j == count-1 {
				break
			}
			value := r.NameFor("zero")
			prefix = append(prefix, Variable(value,
				r.temporaryTypeAt(r.input.Inventory.Checked.TypesInfo.TypeOf(field.Type), field.Type,
					original.End()), nil))
			values = append(values, value)
		}
	}
	return prefix, values
}

// frameSignature reads the checked signature of the function that owns the current return state.
func (r *Context) frameSignature() *types.Signature {
	var signature *types.Signature
	switch function := r.functionState.frame.function.(type) {
	case *ast.FuncDecl:
		signature = r.input.Inventory.Checked.TypesInfo.Defs[function.Name].Type().(*types.Signature)
	case *ast.FuncLit:
		signature = r.input.Inventory.Checked.TypesInfo.TypeOf(function).(*types.Signature)
	}
	return signature
}

// ErrorReturnType reads the current source function's final result type.
func (r *Context) ErrorReturnType() types.Type {
	signature := r.frameSignature()
	return signature.Results().At(signature.Results().Len() - 1).Type()
}

// PrepareFailureResults prepares zero values once in the function that owns the return.
func (r *Context) PrepareFailureResults() []ast.Expr {
	frame := r.functionState.frame
	if !frame.prepared {
		frame.prefix, frame.values = r.prepareReturns(frame.typ)
		frame.prepared = true
	}
	return append([]ast.Expr(nil), frame.values...)
}
