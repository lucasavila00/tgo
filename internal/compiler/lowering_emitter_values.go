package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

func (e *loweringEmitter) emitTypedBind(
	value plannedValue,
	plan *plannedExpression,
	output *ast.BlockStmt,
) ast.Expr {
	expression := e.expression(plan, output)
	if retainTypedBindExpression(value, plan) {
		return expression
	}
	explicit := value.explicit || plan != nil && plan.typ != nil && value.typ != nil &&
		!types.Identical(plan.typ, value.typ)
	return e.emitTypedExpressionBind(value, expression, explicit, output)
}

func retainTypedBindExpression(value plannedValue, plan *plannedExpression) bool {
	if plan != nil && plan.retainContext && !plannedExpressionHasWork(plan) {
		return true
	}
	if plan != nil && isUntypedType(plan.typ) && !plannedExpressionHasWork(plan) {
		return true
	}
	return isUntypedType(value.typ)
}

func (e *loweringEmitter) emitTypedExpressionBind(
	value plannedValue,
	expression ast.Expr,
	explicit bool,
	output *ast.BlockStmt,
) ast.Expr {
	name := e.valueName(value.id, "operand")
	var typeExpression ast.Expr
	if explicit {
		typeExpression = e.contextTypeExpression(value, output)
	}
	if typeExpression != nil {
		output.List = append(output.List, &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{
				Names: []*ast.Ident{name}, Type: typeExpression,
				Values: []ast.Expr{expression},
			}},
		}})
	} else {
		output.List = append(output.List, &ast.AssignStmt{
			Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{expression},
		})
	}
	return ast.NewIdent(name.Name)
}

func packagePath(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	return pkg.Path()
}

func (e *loweringEmitter) expressionResults(
	plan *plannedExpression,
	block *ast.BlockStmt,
) []ast.Expr {
	if plan != nil && plan.resultCount > 1 && plan.work == nil {
		return []ast.Expr{e.expression(plan, block)}
	}
	if plan != nil && plan.work != nil && len(plan.results) != 1 {
		e.operations(plan.work, block)
		result := make([]ast.Expr, 0, len(plan.results))
		for _, value := range plan.results {
			result = append(result, e.valueName(value.id, "result"))
		}
		return result
	}
	return []ast.Expr{e.expression(plan, block)}
}

func (e *loweringEmitter) errorReturn(
	operation *plannedOperation,
	output *ast.BlockStmt,
) ast.Stmt {
	count := e.plan.function.resultType.Len() - 1
	results := make([]ast.Expr, 0, count+1)
	for index := 0; index < count; index++ {
		typ := e.plan.function.resultType.At(index).Type()
		resultType := e.plan.function.resultAST[index]
		if value, ok := e.zeroExpression(typ, resultType, operation.metadata.Bang); ok {
			results = append(results, value)
			continue
		}
		name := e.freshName("zero")
		specification := &ast.ValueSpec{Names: []*ast.Ident{name}, Type: resultType}
		e.unit.generatedValues[specification] = true
		output.List = append(output.List, &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.VAR, Specs: []ast.Spec{specification},
		}})
		results = append(results, name)
	}
	returnedError := ast.Expr(e.valueName(operation.errorValue, "err"))
	if !operation.metadata.Transparent {
		returnedError = call(
			e.unit.generatedObject(e.formatQualifier(), "fmt", "Errorf", operation.metadata.Bang),
			&ast.BasicLit{
				Kind: token.STRING, Value: strconv.Quote(operation.metadata.Name + ": %w"),
			},
			returnedError,
		)
	}
	results = append(results, returnedError)
	return &ast.ReturnStmt{Results: results}
}

func (e *loweringEmitter) zeroExpression(
	value types.Type,
	typeExpression ast.Expr,
	position token.Pos,
) (ast.Expr, bool) {
	underlying := types.Unalias(value)
	if named, ok := underlying.(*types.Named); ok {
		underlying = named.Underlying()
	}
	switch item := underlying.(type) {
	case *types.Basic:
		switch {
		case item.Info()&types.IsBoolean != 0:
			return e.unit.generatedUniverse("false", position), true
		case item.Info()&types.IsString != 0:
			return &ast.BasicLit{Kind: token.STRING, Value: `""`}, true
		case item.Info()&(types.IsInteger|types.IsFloat|types.IsComplex) != 0:
			return &ast.BasicLit{Kind: token.INT, Value: "0"}, true
		case item.Kind() == types.UnsafePointer || item.Kind() == types.UntypedNil:
			return e.unit.generatedUniverse("nil", position), true
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan,
		*types.Signature, *types.Interface:
		return e.unit.generatedUniverse("nil", position), true
	case *types.Array, *types.Struct:
		return &ast.CompositeLit{Type: typeExpression}, true
	}
	return nil, false
}

func (e *loweringEmitter) formatQualifier() string {
	if e.fmtAlias != "" {
		return e.fmtAlias
	}
	for _, specification := range e.source.File.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil || path != "fmt" {
			continue
		}
		if specification.Name == nil {
			e.fmtAlias = "fmt"
			return e.fmtAlias
		}
		if specification.Name.Name == "_" {
			e.fmtAlias = freshASTIdentifier(e.source.File, "fmt")
			if e.fmtAlias == "fmt" {
				specification.Name = nil
			} else {
				specification.Name = ast.NewIdent(e.fmtAlias)
			}
			return e.fmtAlias
		}
		if specification.Name.Name == "." {
			return ""
		}
		e.fmtAlias = specification.Name.Name
		return e.fmtAlias
	}
	e.fmtAlias = freshASTIdentifier(e.source.File, "fmt")
	if e.fmtAlias == "fmt" {
		astutil.AddImport(e.unit.fs, e.source.File, "fmt")
	} else {
		astutil.AddNamedImport(e.unit.fs, e.source.File, e.fmtAlias, "fmt")
	}
	return e.fmtAlias
}

func oneStatement(statements []ast.Stmt) ast.Stmt {
	if len(statements) == 1 {
		return statements[0]
	}
	return &ast.BlockStmt{List: statements}
}

func positionGeneratedStatement(statement ast.Stmt, position token.Pos) {
	switch node := statement.(type) {
	case *ast.AssignStmt:
		positionGeneratedAssignment(node, position)
	case *ast.DeclStmt:
		if declaration, ok := node.Decl.(*ast.GenDecl); ok {
			positionGeneratedDeclaration(declaration, position)
		}
	case *ast.IfStmt:
		positionGeneratedIf(node, position)
	}
}

func positionGeneratedAssignment(statement *ast.AssignStmt, position token.Pos) {
	if statement.TokPos == token.NoPos {
		statement.TokPos = position
	}
	for _, expression := range statement.Lhs {
		positionGeneratedExpression(expression, position)
	}
	for _, expression := range statement.Rhs {
		positionGeneratedExpression(expression, position)
	}
}

func positionGeneratedDeclaration(declaration *ast.GenDecl, position token.Pos) {
	if declaration.TokPos == token.NoPos {
		declaration.TokPos = position
	}
	for _, specification := range declaration.Specs {
		value, ok := specification.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range value.Names {
			positionGeneratedExpression(name, position)
		}
		for _, expression := range value.Values {
			positionGeneratedExpression(expression, position)
		}
	}
}
func positionGeneratedIf(branch *ast.IfStmt, position token.Pos) {
	if branch.If == token.NoPos {
		branch.If = position
	}
	positionGeneratedExpression(branch.Cond, position)
	if branch.Body.Lbrace == token.NoPos {
		branch.Body.Lbrace = position
	}
	if branch.Body.Rbrace == token.NoPos {
		branch.Body.Rbrace = position
	}
	for _, statement := range branch.Body.List {
		if returned, ok := statement.(*ast.ReturnStmt); ok {
			if returned.Return == token.NoPos {
				returned.Return = position
			}
			for _, result := range returned.Results {
				positionGeneratedExpression(result, position)
			}
			continue
		}
		positionGeneratedStatement(statement, position)
	}
}

func positionGeneratedExpression(expression ast.Expr, position token.Pos) {
	if positionGeneratedSpecialExpression(expression, position) {
		return
	}
	switch node := expression.(type) {
	case *ast.Ident:
		if node.NamePos == token.NoPos {
			node.NamePos = position
		}
	case *ast.BasicLit:
		if node.ValuePos == token.NoPos {
			node.ValuePos = position
		}
	case *ast.BinaryExpr:
		if node.OpPos == token.NoPos {
			node.OpPos = position
		}
		positionGeneratedExpression(node.X, position)
		positionGeneratedExpression(node.Y, position)
	case *ast.CallExpr:
		if node.Lparen == token.NoPos {
			node.Lparen = position
		}
		if node.Rparen == token.NoPos {
			node.Rparen = position
		}
		positionGeneratedExpression(node.Fun, position)
		for _, argument := range node.Args {
			positionGeneratedExpression(argument, position)
		}
	case *ast.SelectorExpr:
		positionGeneratedExpression(node.X, position)
		positionGeneratedExpression(node.Sel, position)
	}
}

func positionGeneratedSpecialExpression(expression ast.Expr, position token.Pos) bool {
	switch node := expression.(type) {
	case *ast.IndexExpr:
		positionGeneratedIndex(node.X, []ast.Expr{node.Index}, &node.Lbrack, &node.Rbrack, position)
	case *ast.IndexListExpr:
		positionGeneratedIndex(node.X, node.Indices, &node.Lbrack, &node.Rbrack, position)
	case *ast.CompositeLit:
		if node.Lbrace == token.NoPos {
			node.Lbrace = position
		}
		if node.Rbrace == token.NoPos {
			node.Rbrace = position
		}
	default:
		return false
	}
	return true
}

func positionGeneratedIndex(
	base ast.Expr,
	indices []ast.Expr,
	left *token.Pos,
	right *token.Pos,
	position token.Pos,
) {
	if *left == token.NoPos {
		*left = position
	}
	if *right == token.NoPos {
		*right = position
	}
	positionGeneratedExpression(base, position)
	for _, index := range indices {
		positionGeneratedExpression(index, position)
	}
}
