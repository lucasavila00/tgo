package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (l *propagationLowerer) assignment(node *ast.AssignStmt) []ast.Stmt {
	if !l.assignmentHasLowering(node) {
		return []ast.Stmt{node}
	}
	if statements, fused := l.directShortAssignment(node); fused {
		return statements
	}
	left, prefix := l.assignmentTargets(node.Lhs)
	node.Lhs = left
	if len(node.Rhs) == 1 {
		if metadata, ok := propagationMarker(l.source, node.Rhs[0]); ok {
			values, before := l.propagation(node.Rhs[0])
			prefix = append(prefix, before...)
			if len(values) != len(node.Lhs) {
				l.reportResultCount(metadata, len(values), len(node.Lhs), "assignment")
			}
			node.Rhs = values
			return append(prefix, node)
		}
	}
	values, before := l.expressions(node.Rhs)
	prefix = append(prefix, before...)
	node.Rhs = values
	return append(prefix, node)
}

func (l *propagationLowerer) assignmentHasLowering(node *ast.AssignStmt) bool {
	for _, expression := range append(append([]ast.Expr(nil), node.Lhs...), node.Rhs...) {
		if l.hasLowering(expression) {
			return true
		}
	}
	return false
}

// assignmentTargets evaluates target operands before a moved right side.
// It follows Go assignment order without copying an array target.
func (l *propagationLowerer) assignmentTargets(
	targets []ast.Expr,
) ([]ast.Expr, []ast.Stmt) {
	result := append([]ast.Expr(nil), targets...)
	prefix := []ast.Stmt(nil)
	for index, sourceTarget := range result {
		target, targetPrefix := l.assignmentTarget(sourceTarget)
		result[index] = target
		prefix = append(prefix, targetPrefix...)
	}
	return result, prefix
}

func (l *propagationLowerer) assignmentTarget(
	target ast.Expr,
) (ast.Expr, []ast.Stmt) {
	switch node := target.(type) {
	case *ast.ParenExpr:
		value, prefix := l.assignmentTarget(node.X)
		node.X = value
		return node, prefix
	case *ast.StarExpr:
		value, prefix := l.cheapAssignmentOperand(node.X)
		node.X = value
		return node, prefix
	case *ast.SelectorExpr:
		selection := l.unit.info.Selections[node]
		var value ast.Expr
		var prefix []ast.Stmt
		if selection != nil && selection.Indirect() {
			value, prefix = l.cheapAssignmentOperand(node.X)
		} else {
			value, prefix = l.assignmentTarget(node.X)
		}
		node.X = value
		return node, prefix
	case *ast.IndexExpr:
		var value ast.Expr
		var prefix []ast.Stmt
		if admitsArray(l.unit.info.TypeOf(node.X)) {
			value, prefix = l.assignmentTarget(node.X)
		} else {
			value, prefix = l.cheapAssignmentOperand(node.X)
		}
		index, indexPrefix := l.cheapAssignmentOperand(node.Index)
		node.X, node.Index = value, index
		resultPrefix := make([]ast.Stmt, 0, len(prefix)+len(indexPrefix))
		resultPrefix = append(resultPrefix, prefix...)
		resultPrefix = append(resultPrefix, indexPrefix...)
		return node, resultPrefix
	default:
		return target, nil
	}
}

// admitsArray reports whether a type or type set has an array term.
func admitsArray(value types.Type) bool {
	if value == nil {
		return false
	}
	value = types.Unalias(value)
	switch node := value.(type) {
	case *types.Array:
		return true
	case *types.Named:
		return admitsArray(node.Underlying())
	case *types.TypeParam:
		return admitsArray(node.Constraint())
	case *types.Interface:
		node.Complete()
		for index := range node.NumEmbeddeds() {
			if admitsArray(node.EmbeddedType(index)) {
				return true
			}
		}
	case *types.Union:
		for index := range node.Len() {
			if admitsArray(node.Term(index).Type()) {
				return true
			}
		}
	}
	return false
}

func (l *propagationLowerer) cheapAssignmentOperand(
	expression ast.Expr,
) (ast.Expr, []ast.Stmt) {
	value, prefix := l.expression(expression)
	if isCheapAssignmentOperand(value) {
		return value, prefix
	}
	value, before := l.materialize(value)
	return value, append(prefix, before...)
}

func isCheapAssignmentOperand(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	default:
		return false
	}
}

func (l *propagationLowerer) increment(node *ast.IncDecStmt) []ast.Stmt {
	if !l.hasLowering(node.X) {
		return []ast.Stmt{node}
	}
	value, prefix := l.assignmentTarget(node.X)
	node.X = value
	return append(prefix, node)
}

func (l *propagationLowerer) expressionStatement(node *ast.ExprStmt) []ast.Stmt {
	if metadata, ok := propagationMarker(l.source, node.X); ok {
		values, prefix := l.propagation(node.X)
		if len(values) != 0 {
			l.unit.failAt(metadata.Bang, "discarded propagated call returns values")
		}
		return prefix
	}
	value, prefix := l.expression(node.X)
	node.X = value
	return append(prefix, node)
}

func (l *propagationLowerer) declaration(node *ast.DeclStmt) []ast.Stmt {
	general, ok := node.Decl.(*ast.GenDecl)
	if !ok {
		return []ast.Stmt{node}
	}
	if statements, split := l.splitVariableDeclaration(general); split {
		return statements
	}
	if statements, fused := l.directVariableDeclaration(node, general); fused {
		return statements
	}
	prefix := []ast.Stmt(nil)
	for _, item := range general.Specs {
		value, ok := item.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if len(value.Values) == 1 {
			if metadata, direct := propagationMarker(l.source, value.Values[0]); direct {
				values, before := l.propagation(value.Values[0])
				if len(values) != len(value.Names) {
					l.reportResultCount(metadata, len(values), len(value.Names), "declaration")
				}
				value.Values = values
				prefix = append(prefix, before...)
				continue
			}
		}
		values, before := l.expressions(value.Values)
		value.Values = values
		prefix = append(prefix, before...)
	}
	return append(prefix, node)
}

// splitVariableDeclaration keeps lowering work between its source specifications.
func (l *propagationLowerer) splitVariableDeclaration(
	general *ast.GenDecl,
) ([]ast.Stmt, bool) {
	if general.Tok != token.VAR || len(general.Specs) < 2 ||
		!l.variableDeclarationHasLowering(general) {
		return nil, false
	}
	result := []ast.Stmt(nil)
	for index, specification := range general.Specs {
		if value, ok := specification.(*ast.ValueSpec); ok && value.Doc != nil {
			for _, comment := range value.Doc.List {
				comment.Slash = specification.Pos() - 1
			}
		}
		declaration := &ast.GenDecl{
			TokPos: specification.Pos(),
			Tok:    general.Tok,
			Specs:  []ast.Spec{specification},
		}
		if index == 0 {
			declaration.Doc = general.Doc
		}
		statements := l.declaration(&ast.DeclStmt{Decl: declaration})
		declarationIndex := 0
		for index, statement := range statements {
			item, ok := statement.(*ast.DeclStmt)
			if ok && item.Decl == declaration {
				declarationIndex = index
				break
			}
			positionGeneratedStatement(statement, specification.Pos())
		}
		positionGeneratedStatement(statements[declarationIndex], specification.Pos())
		position := specification.End()
		if value, ok := specification.(*ast.ValueSpec); ok && value.Comment != nil {
			position = value.Comment.End() + 1
		}
		for _, statement := range statements[declarationIndex+1:] {
			positionGeneratedStatement(statement, position)
		}
		result = append(result, statements...)
	}
	return result, true
}

func (l *propagationLowerer) variableDeclarationHasLowering(
	general *ast.GenDecl,
) bool {
	for _, specification := range general.Specs {
		value, ok := specification.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, expression := range value.Values {
			if l.hasLowering(expression) {
				return true
			}
		}
	}
	return false
}

func (l *propagationLowerer) reportResultCount(
	metadata propagationSource,
	results int,
	targets int,
	context string,
) {
	l.unit.failAt(
		metadata.Bang,
		"propagated call returns %d values; %s needs %d",
		results,
		context,
		targets,
	)
}
