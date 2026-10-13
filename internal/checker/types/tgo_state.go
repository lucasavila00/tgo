package types

import "go/ast"

// tgoCheckState belongs to one Checker. Cached facts remain available when
// checking revisits expressions for assignment and generic inference.
// statementContext stays in Go's environment so function entry and exit save
// it together with the source scope and signature.
type tgoCheckState struct {
	// Facts for source nodes and marker names.
	conditionalValues   map[*ast.CallExpr][]*operand
	expressionTypes     map[ast.Expr]Type
	comprehensionRanges map[*ast.RangeStmt]bool
	comprehensionTypes  map[string][]Type
	inferConditionals   map[*ast.CallExpr]bool

	// Active cases are removed after checking their value switch.
	conditionalCases map[*ast.CaseClause]*conditionalCase
	// builtinArguments saves and restores this context for nested calls.
	sliceSourceContext Type
}
