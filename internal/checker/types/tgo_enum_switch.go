package types

import (
	"strings"

	"github.com/lucasavila00/tgo/internal/ast"
	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// enumSwitch checks case identities and bindings after the switch input is checked.
func (check *Checker) enumSwitch(
	statement *ast.TypeSwitchStmt, context stmtContext, binding *ast.Ident, input *operand, delayedTop int,
) bool {
	enum := EnumOf(input.typ())
	if enum == nil {
		return false
	}
	if check.EnumSwitches == nil {
		check.EnumSwitches = make(map[*ast.TypeSwitchStmt]*Enum)
		check.EnumCases = make(map[*ast.CaseClause][]*EnumVariant)
	}
	check.EnumSwitches[statement] = enum
	seen := make(map[*EnumVariant]ast.Expr)
	var variables []*Var
	var exhaustive, defaultCase *ast.CaseClause
	for _, node := range statement.Body.List {
		clause := node.(*ast.CaseClause)
		if clause.Exhaustive {
			check.enumExhaustiveLabel(clause, exhaustive)
			exhaustive = clause
			continue
		}
		if clause.List == nil {
			if defaultCase != nil {
				check.error(clause, DuplicateDefault, "multiple default cases in enum switch")
			}
			defaultCase = clause
		}
		caseType := check.enumCaseTypes(clause, enum, input.typ(), seen)
		variable := check.enumCaseBody(clause, context, binding, caseType)
		if variable != nil {
			variables = append(variables, variable)
		}
	}
	if exhaustive != nil {
		check.enumCoverage(enum, exhaustive, defaultCase, seen)
	}
	check.enumSwitchUsage(binding, variables, delayedTop)
	return true
}

// enumExhaustiveLabel requires one empty exhaustive label.
func (check *Checker) enumExhaustiveLabel(clause, previous *ast.CaseClause) {
	if previous != nil {
		check.error(clause, InvalidTypeSwitch, "multiple exhaustive labels in enum switch")
	}
	for _, statement := range clause.Body {
		if _, empty := statement.(*ast.EmptyStmt); !empty {
			check.error(statement, InvalidTypeSwitch, "exhaustive label must have no statements")
			return
		}
	}
}

// enumCaseTypes records variant identities and selects the type of the case binding.
func (check *Checker) enumCaseTypes(
	clause *ast.CaseClause, enum *Enum, input Type, seen map[*EnumVariant]ast.Expr,
) Type {
	var variants []*EnumVariant
	for _, expression := range clause.List {
		variant, recognized := check.enumVariant(expression)
		if !recognized {
			check.error(expression, InvalidTypeSwitch, "enum case must name a declared variant")
			continue
		}
		if variant == nil {
			continue
		}
		if variant.enum != enum {
			check.errorf(expression, InvalidTypeSwitch, "variant %s belongs to %s, not %s",
				variant.name, variant.enum.named, enum.named)
			continue
		}
		if previous := seen[variant]; previous != nil {
			check.errorf(expression, DuplicateCase, "duplicate enum case %s", variant.name)
		}
		seen[variant] = expression
		variants = append(variants, variant)
	}
	check.EnumCases[clause] = variants
	if len(clause.List) == 1 && len(variants) == 1 {
		return variants[0].typ
	}
	return input
}

// enumCaseBody checks one branch with the same local-binding rules as a Go type switch.
func (check *Checker) enumCaseBody(
	clause *ast.CaseClause, context stmtContext, binding *ast.Ident, typ Type,
) *Var {
	check.openScope(clause, "enum case")
	defer check.closeScope()
	var variable *Var
	if binding != nil {
		variable = newVar(LocalVar, binding.Pos(), check.pkg, binding.Name, typ)
		check.declare(check.scope, nil, variable, clause.Colon)
		check.recordImplicit(clause, variable)
	}
	if branch := check.tgo.conditionalCases[clause]; branch != nil {
		branch.values = check.conditionalCase(clause, branch.context)
	} else {
		check.stmtList(context, clause.Body)
	}
	return variable
}

// enumCoverage reports missing cases in declaration order.
func (check *Checker) enumCoverage(
	enum *Enum, exhaustive, defaultCase *ast.CaseClause, seen map[*EnumVariant]ast.Expr,
) {
	if defaultCase != nil {
		check.error(exhaustive, InvalidTypeSwitch, "enum switch cannot use both default and exhaustive")
	}
	var missing []string
	for _, variant := range enum.variants {
		if seen[variant] == nil {
			missing = append(missing, variant.name)
		}
	}
	if len(missing) != 0 {
		check.errorf(exhaustive, InvalidTypeSwitch, "exhaustive enum switch is missing: %s", strings.Join(missing, ", "))
	}
}

// enumSwitchUsage counts uses in deferred closure bodies before checking the shared binding name.
func (check *Checker) enumSwitchUsage(binding *ast.Ident, variables []*Var, delayedTop int) {
	if binding == nil {
		return
	}
	check.processDelayed(delayedTop)
	used := false
	for _, variable := range variables {
		used = used || check.usedVars[variable]
		check.usedVars[variable] = true
	}
	if !used {
		check.softErrorf(binding, UnusedVar, "%s declared and not used", binding.Name)
	}
}

// rejectExhaustive reports an enum-only label used in an ordinary switch.
func (check *Checker) rejectExhaustive(clause *ast.CaseClause) bool {
	if !clause.Exhaustive {
		return false
	}
	check.error(clause, InvalidTypeSwitch, "exhaustive requires an enum type switch")
	return true
}
