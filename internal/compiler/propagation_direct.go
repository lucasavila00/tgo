package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

// directShortAssignment keeps source names in the propagated Go call.
func (l *propagationLowerer) directShortAssignment(
	node *ast.AssignStmt,
) ([]ast.Stmt, bool) {
	if node.Tok != token.DEFINE || len(node.Rhs) != 1 || !identifierTargets(node.Lhs) {
		return nil, false
	}
	signature := l.directPropagationSignature(node.Rhs[0])
	call, statements, callIndex, direct := l.directCall(
		node.Rhs[0], len(node.Lhs), "assignment",
	)
	if !direct {
		return nil, false
	}
	if call == nil {
		return append(statements, node), true
	}
	l.rememberDirectResultTypes(node.Lhs, signature)
	node.Lhs = append(node.Lhs, call.Lhs[len(call.Lhs)-1])
	node.Rhs = call.Rhs
	statements[callIndex] = node
	return statements, true
}

// directPropagationSignature gets the source call signature for one direct marker.
func (l *propagationLowerer) directPropagationSignature(
	expression ast.Expr,
) *types.Signature {
	marker, ok := expression.(*ast.CallExpr)
	if !ok || len(marker.Args) != 1 {
		return nil
	}
	call, ok := unwrappedCompilerCall(marker.Args[0])
	if !ok {
		return nil
	}
	return l.propagationCallSignature(call.Fun)
}

// rememberDirectResultTypes records names inferred by an earlier propagated call.
func (l *propagationLowerer) rememberDirectResultTypes(
	targets []ast.Expr,
	signature *types.Signature,
) {
	if signature == nil || signature.Results().Len() <= len(targets) {
		return
	}
	for index, target := range targets {
		identifier, ok := target.(*ast.Ident)
		if !ok || identifier.Name == "_" {
			continue
		}
		l.rememberDirectResultType(identifier, index, signature)
	}
}

func (l *propagationLowerer) rememberDirectResultNames(
	targets []*ast.Ident,
	signature *types.Signature,
) {
	if signature == nil || signature.Results().Len() <= len(targets) {
		return
	}
	for index, identifier := range targets {
		if identifier.Name != "_" {
			l.rememberDirectResultType(identifier, index, signature)
		}
	}
}

func (l *propagationLowerer) rememberDirectResultType(
	identifier *ast.Ident,
	index int,
	signature *types.Signature,
) {
	typ := signature.Results().At(index).Type()
	object := l.unit.info.Defs[identifier]
	if object != nil {
		l.inferredResultTypes[object] = typ
	}
	l.inferredResultNames[identifier.Name] = typ
}

// rememberSimpleAssignmentTypes carries inferred types through an ordinary declaration.
func (l *propagationLowerer) rememberSimpleAssignmentTypes(node *ast.AssignStmt) {
	if node.Tok != token.DEFINE || len(node.Lhs) != len(node.Rhs) {
		return
	}
	for index, target := range node.Lhs {
		identifier, ok := target.(*ast.Ident)
		if !ok || identifier.Name == "_" {
			continue
		}
		object := l.unit.info.Defs[identifier]
		typ := l.inferredExpressionType(node.Rhs[index])
		if object != nil && typ != nil {
			l.inferredResultTypes[object] = typ
		}
		if typ != nil {
			l.inferredResultNames[identifier.Name] = typ
		}
	}
}

// inferredExpressionType gets a type that depends on an earlier propagated result.
func (l *propagationLowerer) inferredExpressionType(expression ast.Expr) types.Type {
	if typ := l.unit.info.TypeOf(expression); typ != nil {
		basic, invalid := types.Unalias(typ).(*types.Basic)
		if !invalid || basic.Kind() != types.Invalid {
			return typ
		}
	}
	switch node := expression.(type) {
	case *ast.ParenExpr:
		return l.inferredExpressionType(node.X)
	case *ast.Ident:
		typ := l.inferredResultTypes[l.unit.info.ObjectOf(node)]
		if typ == nil {
			typ = l.inferredResultNames[node.Name]
		}
		return typ
	case *ast.IndexExpr:
		container := l.inferredExpressionType(node.X)
		if container == nil {
			return nil
		}
		container = types.Unalias(container).Underlying()
		switch value := container.(type) {
		case *types.Array:
			return value.Elem()
		case *types.Slice:
			return value.Elem()
		case *types.Map:
			return value.Elem()
		case *types.Pointer:
			array, _ := types.Unalias(value.Elem()).Underlying().(*types.Array)
			if array != nil {
				return array.Elem()
			}
		}
	}
	return nil
}

// directVariableDeclaration keeps an inferred var declaration as one Go call.
func (l *propagationLowerer) directVariableDeclaration(
	node *ast.DeclStmt,
	general *ast.GenDecl,
) ([]ast.Stmt, bool) {
	if general.Tok != token.VAR || len(general.Specs) != 1 {
		return nil, false
	}
	value, ok := general.Specs[0].(*ast.ValueSpec)
	if !ok || value.Type != nil || len(value.Values) != 1 {
		return nil, false
	}
	signature := l.directPropagationSignature(value.Values[0])
	call, statements, callIndex, direct := l.directCall(
		value.Values[0], len(value.Names), "declaration",
	)
	if !direct {
		return nil, false
	}
	if call == nil {
		return append(statements, node), true
	}
	errorName, ok := call.Lhs[len(call.Lhs)-1].(*ast.Ident)
	if !ok {
		return append(statements, node), true
	}
	errorName.NamePos = value.Names[len(value.Names)-1].End()
	l.rememberDirectResultNames(value.Names, signature)
	value.Names = append(value.Names, errorName)
	value.Values = call.Rhs
	statements[callIndex] = node
	if value.Comment != nil {
		for _, statement := range statements[callIndex+1:] {
			positionGeneratedStatement(statement, value.Comment.End()+1)
		}
	}
	return statements, true
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
		positionGeneratedBranchStatement(statement, position)
	}
}

func positionGeneratedBranchStatement(statement ast.Stmt, position token.Pos) {
	if returned, ok := statement.(*ast.ReturnStmt); ok {
		if returned.Return == token.NoPos {
			returned.Return = position
		}
		for _, result := range returned.Results {
			positionGeneratedExpression(result, position)
		}
		return
	}
	positionGeneratedStatement(statement, position)
}

func positionGeneratedExpression(expression ast.Expr, position token.Pos) {
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
	case *ast.CompositeLit:
		if node.Lbrace == token.NoPos {
			node.Lbrace = position
		}
		if node.Rbrace == token.NoPos {
			node.Rbrace = position
		}
	}
}

// directCall returns the call assignment and its error branch for a direct marker.
func (l *propagationLowerer) directCall(
	expression ast.Expr,
	targets int,
	context string,
) (*ast.AssignStmt, []ast.Stmt, int, bool) {
	metadata, direct := propagationMarker(l.source, expression)
	if !direct {
		return nil, nil, -1, false
	}
	values, statements := l.propagation(expression)
	if len(values) != targets {
		l.reportResultCount(metadata, len(values), targets, context)
		return nil, statements, -1, true
	}
	callIndex := len(statements) - 2
	if callIndex < 0 {
		return nil, statements, -1, true
	}
	callAssignment, ok := statements[callIndex].(*ast.AssignStmt)
	if !ok || len(callAssignment.Lhs) != targets+1 {
		return nil, statements, -1, true
	}
	for _, value := range values {
		if name, ok := value.(*ast.Ident); ok {
			delete(l.names, name.Name)
		}
	}
	return callAssignment, statements, callIndex, true
}

func identifierTargets(expressions []ast.Expr) bool {
	for _, expression := range expressions {
		if _, ok := expression.(*ast.Ident); !ok {
			return false
		}
	}
	return true
}
