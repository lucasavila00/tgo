package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

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
			var changes []edit
			changes, failure = lowerExhaustiveClause(
				files, file, switchStatement, clause, used, storedReceivers,
			)
			if failure != nil {
				return false
			}
			edits = append(edits, changes...)
		}
		return true
	})
	return edits, storedReceivers, failure
}

func lowerExhaustiveClause(
	files *token.FileSet,
	file *token.File,
	switched *syntax.SwitchStatement,
	clause *syntax.CaseClause,
	used map[string]bool,
	storedReceivers map[string]string,
) ([]edit, error) {
	if len(clause.Body) != 0 {
		return nil, fmt.Errorf(
			"%s: exhaustive clause must not have a body",
			files.Position(clause.Case),
		)
	}
	receiver, ok := exhaustiveReceiver(switched.Tag)
	if !ok {
		return nil, fmt.Errorf(
			"%s: exhaustive clause requires switch value.Tag()",
			files.Position(clause.Case),
		)
	}
	marker := freshIdentifier("tgoExhaustive", used)
	storedReceivers[marker] = ""
	if exhaustiveReceiverNeedsStorage(receiver) {
		storedReceivers[marker] = freshIdentifier("enumValue", used)
	}
	start := file.Offset(clause.Exhaustive)
	colon := file.Offset(clause.Colon) + 1
	return []edit{
		{start: start, end: start + len(exhaustiveWord), text: "default"},
		{
			start: colon, end: colon,
			text: " panic(" + marker + "()) " + enumDefaultComment,
		},
	}, nil
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
	panicCall *ast.CallExpr
	receiver  ast.Expr
	selector  *ast.SelectorExpr
	name      string
}

// lowerExhaustiveReceiverEvaluations stores non-local switch receivers once.
func lowerExhaustiveReceiverEvaluations(
	file *ast.File,
	names map[string]string,
) []exhaustiveDefault {
	if len(names) == 0 {
		return nil
	}
	var defaults []exhaustiveDefault
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
			defaults = append(defaults, exhaustiveDefault{
				panicCall: plan.panicCall, tag: node.Tag,
			})
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
			defaults = append(defaults, exhaustiveDefault{
				panicCall: plan.panicCall, tag: switched.Tag,
			})
			if plan.name == "" {
				applyExhaustivePlan(plan)
				return true
			}
			initializer := switched.Init
			switched.Init = nil
			applyExhaustivePlan(plan)
			node.Stmt = &ast.SwitchStmt{Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.CaseClause{Body: []ast.Stmt{
					initializer, exhaustiveReceiverAssignment(plan), switched,
				}},
			}}}
		}
		return true
	})
	return defaults
}

type exhaustiveDefault struct {
	panicCall *ast.CallExpr
	tag       ast.Expr
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
			panicCall: panicCall,
			receiver:  selector.X,
			selector:  selector,
			name:      name,
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
	if plan.name == "" {
		return switched
	}
	assignment := exhaustiveReceiverAssignment(plan)
	if initializer == nil {
		switched.Init = assignment
		return switched
	}
	switched.Init = nil
	return &ast.BlockStmt{List: []ast.Stmt{initializer, assignment, switched}}
}

func applyExhaustivePlan(plan exhaustiveReceiverPlan) {
	if plan.name != "" {
		tagReceiver := ast.NewIdent(plan.name)
		tagReceiver.NamePos = plan.receiver.Pos()
		plan.selector.X = tagReceiver
	}
	plan.panicCall.Args[0] = &ast.BasicLit{
		Kind: token.STRING, Value: `"invalid enum tag"`,
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

// setExhaustiveDefaultMessages gives each generated panic its enum name.
func (p *packageUnit) setExhaustiveDefaultMessages() {
	for _, source := range p.Sources {
		for _, generated := range source.Exhaustive {
			tag, ok := types.Unalias(p.info.TypeOf(generated.tag)).(*types.Named)
			if !ok {
				continue
			}
			name, ok := strings.CutSuffix(tag.Obj().Name(), "Tag")
			if !ok || name == "" {
				continue
			}
			generated.panicCall.Args[0] = &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("%q", "invalid "+name+" tag"),
			}
		}
	}
}
