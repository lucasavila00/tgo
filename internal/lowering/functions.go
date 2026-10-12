package lowering

import (
	"go/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

type returnFrame struct {
	function ast.Node
	typ      *ast.FuncType
	prepared bool
	prefix   []ast.Stmt
	values   []ast.Expr
}

func (r *rewrite) function(node ast.Node, typ *ast.FuncType, body *ast.BlockStmt) (*ast.FuncType,
	*ast.BlockStmt) {
	functionRewrite := &rewrite{
		input:         r.input,
		request:       r.request,
		file:          r.file,
		functionState: &functionState{frame: &returnFrame{function: node, typ: typ}, hasGoto: r.functionHasGoto(body)},
	}
	output := functionRewrite.block(body)
	if functionRewrite.functionState.frame.prepared {
		output.List = append(functionRewrite.functionState.frame.prefix, output.List...)
	}
	if functionRewrite.input.source.inventory.Checked.TypesInfo.TGoReturnBlocks[body] {
		output = functionRewrite.returnBlock(typ, output)
	}
	if aliases := functionRewrite.file.aliasStatements[node]; len(aliases) != 0 {
		output.List = append(aliases, output.List...)
	}
	return typ, output
}

// prepareReturns saves zero values for the results before the error.
func (r *rewrite) prepareReturns(original *ast.FuncType) []ast.Stmt {
	if original.Results == nil {
		panic("propagation requires an error return")
	}
	var prefix []ast.Stmt
	for i, field := range original.Results.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for j := 0; j < count; j++ {
			if i == len(original.Results.List)-1 && j == count-1 {
				break
			}
			value := r.nameFor("zero")
			prefix = append(prefix, variable(value,
				r.temporaryTypeAt(r.input.source.inventory.Checked.TypesInfo.TypeOf(field.Type), field.Type,
					original.End()), nil))
			r.functionState.frame.values = append(r.functionState.frame.values, value)
		}
	}
	return prefix
}

func (r *rewrite) frameSignature() *types.Signature {
	var signature *types.Signature
	switch function := r.functionState.frame.function.(type) {
	case *ast.FuncDecl:
		signature = r.input.source.inventory.Checked.TypesInfo.Defs[function.Name].Type().(*types.Signature)
	case *ast.FuncLit:
		signature = r.input.source.inventory.Checked.TypesInfo.TypeOf(function).(*types.Signature)
	}
	return signature
}

// errorReturnType reads the current source function's final result type.
func (r *rewrite) errorReturnType() types.Type {
	signature := r.frameSignature()
	return signature.Results().At(signature.Results().Len() - 1).Type()
}

// failureResults prepares zero values once in the function that owns the return.
func (r *rewrite) failureResults() []ast.Expr {
	frame := r.functionState.frame
	if !frame.prepared {
		frame.prefix = r.prepareReturns(frame.typ)
		frame.prepared = true
	}
	return append([]ast.Expr(nil), frame.values...)
}
