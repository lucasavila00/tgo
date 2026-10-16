package types

import "github.com/lucasavila00/tgo/internal/ast"

// tgoCheckState belongs to one Checker. Cached facts remain available when
// checking revisits expressions for assignment and generic inference.
// statementContext stays in Go's environment so function entry and exit save
// it together with the source scope and signature.
type tgoCheckState struct {
	enumConstructors map[Object]*EnumVariant
	enums            map[*ast.EnumType]*Enum
	packageScopes    []*Scope // local TGo scopes checked after closure bodies

	// Facts for source expressions.
	conditionalValues map[ast.Expr][]*operand
	expressionTypes   map[ast.Expr]Type
	inferConditionals map[ast.Expr]bool

	// Active cases are removed after checking their value switch.
	conditionalCases map[*ast.CaseClause]*conditionalCase
	// builtinArguments saves and restores this context for nested calls.
	sliceSourceContext Type
}
