package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

func (l *propagationLowerer) errorBranch(
	metadata propagationSource,
	errorName *ast.Ident,
) ast.Stmt {
	zeroValues, body := l.zeroReturnValues(metadata.Bang, len(l.function.resultAST)-1)
	returnedError := ast.Expr(errorName)
	if !metadata.Transparent {
		formatError := l.unit.generatedObject(
			l.formatQualifier(), "fmt", "Errorf", metadata.Bang,
		)
		returnedError = call(formatError,
			&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(metadata.Name + ": %w")},
			errorName,
		)
	}
	results := make([]ast.Expr, 0, len(zeroValues)+1)
	results = append(results, zeroValues...)
	results = append(results, returnedError)
	body = append(body, &ast.ReturnStmt{Results: results})
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: errorName, Op: token.NEQ,
			Y: l.unit.generatedUniverse("nil", metadata.Bang),
		},
		Body: &ast.BlockStmt{List: body},
	}
}

// failureReturn adds one zero value for each leading comma.
func (l *propagationLowerer) failureReturn(
	statement *ast.ReturnStmt,
	prefix []ast.Stmt,
	commas []token.Pos,
) []ast.Stmt {
	comma := commas[0]
	if len(statement.Results) != 1 {
		l.unit.failAt(comma, "failure return error expression must produce one value")
		return append(prefix, statement)
	}
	if len(commas) >= l.function.resultType.Len() {
		excess := l.function.resultType.Len() - 1
		if excess < 0 {
			excess = 0
		}
		l.unit.failAt(
			commas[excess],
			"failure return has more commas than preceding results",
		)
		return append(prefix, statement)
	}
	last := l.function.resultType.Len() - 1
	if !isPredeclaredError(l.function.resultType.At(last).Type()) {
		l.unit.failAt(comma, "failure return function must end in the Go error type")
		return append(prefix, statement)
	}
	zeroValues, declarations := l.zeroReturnValues(comma, len(commas))
	zeroValues = append(zeroValues, statement.Results[0])
	statement.Results = zeroValues
	prefix = append(prefix, declarations...)
	return append(prefix, statement)
}

// zeroReturnValues makes exact zero values for the requested leading results.
func (l *propagationLowerer) zeroReturnValues(
	position token.Pos,
	count int,
) ([]ast.Expr, []ast.Stmt) {
	zeroValues := make([]ast.Expr, 0, count)
	declarations := make([]ast.Stmt, 0, count)
	for index, resultType := range l.function.resultAST[:count] {
		valueType := l.function.resultType.At(index).Type()
		if value, ok := l.zeroExpression(
			valueType,
			resultType,
			position,
		); ok {
			zeroValues = append(zeroValues, value)
			continue
		}
		name := l.freshName("zero")
		if parameter, ok := types.Unalias(valueType).(*types.TypeParam); ok {
			resultType = ast.NewIdent(parameter.Obj().Name())
		}
		specification := &ast.ValueSpec{Names: []*ast.Ident{name}, Type: resultType}
		l.unit.generatedValues[specification] = true
		declaration := &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok:   token.VAR,
			Specs: []ast.Spec{specification},
		}}
		declarations = append(declarations, declaration)
		zeroValues = append(zeroValues, name)
	}
	return zeroValues, declarations
}

func (l *propagationLowerer) formatQualifier() string {
	if l.fmtAlias != "" {
		return l.fmtAlias
	}
	for _, specification := range l.source.File.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil || path != "fmt" {
			continue
		}
		if specification.Name == nil {
			l.fmtAlias = "fmt"
			return l.fmtAlias
		}
		if specification.Name.Name == "_" {
			l.fmtAlias = freshASTIdentifier(l.source.File, "fmt")
			if l.fmtAlias == "fmt" {
				specification.Name = nil
			} else {
				specification.Name = ast.NewIdent(l.fmtAlias)
			}
			return l.fmtAlias
		}
		if specification.Name.Name == "." {
			return ""
		}
		l.fmtAlias = specification.Name.Name
		return l.fmtAlias
	}
	l.fmtAlias = freshASTIdentifier(l.source.File, "fmt")
	if l.fmtAlias == "fmt" {
		astutil.AddImport(l.unit.fs, l.source.File, "fmt")
	} else {
		astutil.AddNamedImport(l.unit.fs, l.source.File, l.fmtAlias, "fmt")
	}
	return l.fmtAlias
}
