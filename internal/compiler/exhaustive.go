package compiler

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"

	"tgo/pkg/syntax"
)

const exhaustiveWord = "exhaustive"

const enumDefaultComment = "// unreachable: tgolint requires a case per tag"

// lowerExhaustiveClauses emits the Go default for each exhaustive clause.
func lowerExhaustiveClauses(
	files *token.FileSet,
	file *token.File,
	tree *syntax.File,
	data []byte,
	used map[string]bool,
	edits []edit,
) ([]edit, map[string]string, error) {
	var failure error
	storedReceivers := make(map[string]string)
	syntax.Inspect(tree, func(node *syntax.Node) bool {
		if failure != nil {
			return false
		}
		statement, ok := syntax.StatementOf(node)
		if !ok || statement.Tag() != syntax.StatementTagSwitch {
			return true
		}
		switchStatement := statement.SwitchPayload().Value
		for _, clauseStatement := range switchStatement.Body.List {
			if clauseStatement.Tag() != syntax.StatementTagCase {
				continue
			}
			clause := clauseStatement.CasePayload().Value
			if !clause.Exhaustive.IsValid() {
				continue
			}
			start := file.Offset(clause.Exhaustive)
			if len(clause.Body) != 0 {
				failure = fmt.Errorf(
					"%s: exhaustive clause must not have a body",
					files.Position(clause.Case),
				)
				return false
			}
			receiver, ok := exhaustiveReceiver(switchStatement.Tag)
			if !ok {
				failure = fmt.Errorf(
					"%s: exhaustive clause requires switch value.Tag()",
					files.Position(clause.Case),
				)
				return false
			}
			receiverStart := file.Offset(syntax.ExpressionPosition(receiver))
			receiverEnd := file.Offset(syntax.ExpressionEnd(receiver))
			if receiverStart < 0 || receiverEnd < receiverStart || receiverEnd > len(data) {
				failure = fmt.Errorf(
					"%s: exhaustive clause has an invalid switch receiver",
					files.Position(clause.Case),
				)
				return false
			}
			receiverText := string(data[receiverStart:receiverEnd])
			unknownTag := receiverText + ".UnknownTag()"
			if exhaustiveReceiverNeedsStorage(receiver) {
				marker := freshIdentifier("tgoExhaustive", used)
				storedReceivers[marker] = freshIdentifier("enumValue", used)
				unknownTag = marker + "()"
			}
			edits = append(edits,
				edit{start: start, end: start + len(exhaustiveWord), text: "default"},
				edit{
					start: file.Offset(clause.Colon) + 1,
					end:   file.Offset(clause.Colon) + 1,
					text:  " panic(" + unknownTag + ") " + enumDefaultComment,
				},
			)
		}
		return true
	})
	return edits, storedReceivers, failure
}

func exhaustiveReceiverNeedsStorage(expression *syntax.Expression) bool {
	for expression != nil && expression.Tag() == syntax.ExpressionTagParenthesized {
		expression = expression.ParenthesizedPayload().Value.Expression
	}
	return expression == nil || expression.Tag() != syntax.ExpressionTagIdentifier
}

func exhaustiveReceiver(expression *syntax.Expression) (*syntax.Expression, bool) {
	for expression != nil && expression.Tag() == syntax.ExpressionTagParenthesized {
		expression = expression.ParenthesizedPayload().Value.Expression
	}
	if expression == nil || expression.Tag() != syntax.ExpressionTagCall {
		return nil, false
	}
	call := expression.CallPayload().Value
	if len(call.Args) != 0 || call.Callee.Tag() != syntax.ExpressionTagSelector {
		return nil, false
	}
	selector := call.Callee.SelectorPayload().Value
	if selector.Selector.Name != "Tag" {
		return nil, false
	}
	return selector.Expression, true
}

type exhaustiveReceiverPlan struct {
	defaultCall *ast.CallExpr
	receiver    ast.Expr
	selector    *ast.SelectorExpr
	name        string
}

// lowerExhaustiveReceiverEvaluations stores non-local switch receivers once.
func lowerExhaustiveReceiverEvaluations(file *ast.File, names map[string]string) {
	if len(names) == 0 {
		return
	}
	astutil.Apply(file, nil, func(cursor *astutil.Cursor) bool {
		switch node := cursor.Node().(type) {
		case *ast.SwitchStmt:
			plan, ok := exhaustivePlan(node, names)
			if !ok {
				return true
			}
			if node.Init != nil {
				if _, labeled := cursor.Parent().(*ast.LabeledStmt); labeled {
					return true
				}
			}
			cursor.Replace(lowerExhaustiveSwitch(node, plan))
		case *ast.LabeledStmt:
			switched, ok := node.Stmt.(*ast.SwitchStmt)
			if !ok || switched.Init == nil {
				return true
			}
			plan, ok := exhaustivePlan(switched, names)
			if !ok {
				return true
			}
			initializer := switched.Init
			switched.Init = nil
			applyExhaustivePlan(plan)
			node.Stmt = switched
			cursor.Replace(&ast.BlockStmt{List: []ast.Stmt{
				initializer, exhaustiveReceiverAssignment(plan), node,
			}})
		}
		return true
	})
}

func exhaustivePlan(
	switched *ast.SwitchStmt,
	names map[string]string,
) (exhaustiveReceiverPlan, bool) {
	selector, ok := exhaustiveTagSelector(switched.Tag)
	if !ok {
		return exhaustiveReceiverPlan{}, false
	}
	for _, statement := range switched.Body.List {
		clause, ok := statement.(*ast.CaseClause)
		if !ok || len(clause.List) != 0 || len(clause.Body) != 1 {
			continue
		}
		expression, ok := clause.Body[0].(*ast.ExprStmt)
		if !ok {
			continue
		}
		panicCall, ok := expression.X.(*ast.CallExpr)
		if !ok || len(panicCall.Args) != 1 {
			continue
		}
		panicName, ok := panicCall.Fun.(*ast.Ident)
		if !ok || panicName.Name != "panic" {
			continue
		}
		defaultCall, ok := panicCall.Args[0].(*ast.CallExpr)
		if !ok || len(defaultCall.Args) != 0 {
			continue
		}
		marker, ok := defaultCall.Fun.(*ast.Ident)
		if !ok {
			continue
		}
		name, ok := names[marker.Name]
		if !ok {
			continue
		}
		return exhaustiveReceiverPlan{
			defaultCall: defaultCall,
			receiver:    selector.X,
			selector:    selector,
			name:        name,
		}, true
	}
	return exhaustiveReceiverPlan{}, false
}

func exhaustiveTagSelector(expression ast.Expr) (*ast.SelectorExpr, bool) {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parenthesized.X
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return nil, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return selector, ok && selector.Sel.Name == "Tag"
}

func lowerExhaustiveSwitch(
	switched *ast.SwitchStmt,
	plan exhaustiveReceiverPlan,
) ast.Stmt {
	initializer := switched.Init
	applyExhaustivePlan(plan)
	assignment := exhaustiveReceiverAssignment(plan)
	if initializer == nil {
		switched.Init = assignment
		return switched
	}
	switched.Init = nil
	return &ast.BlockStmt{List: []ast.Stmt{initializer, assignment, switched}}
}

func applyExhaustivePlan(plan exhaustiveReceiverPlan) {
	tagReceiver := ast.NewIdent(plan.name)
	tagReceiver.NamePos = plan.receiver.Pos()
	plan.selector.X = tagReceiver
	plan.defaultCall.Fun = &ast.SelectorExpr{
		X: ast.NewIdent(plan.name), Sel: ast.NewIdent("UnknownTag"),
	}
}

func exhaustiveReceiverAssignment(plan exhaustiveReceiverPlan) *ast.AssignStmt {
	name := ast.NewIdent(plan.name)
	name.NamePos = plan.receiver.Pos()
	return &ast.AssignStmt{
		Lhs:    []ast.Expr{name},
		TokPos: plan.receiver.Pos(),
		Tok:    token.DEFINE,
		Rhs:    []ast.Expr{plan.receiver},
	}
}
