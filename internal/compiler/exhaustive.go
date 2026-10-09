package compiler

import (
	"fmt"
	"go/token"

	"tgo/pkg/syntax"
)

const exhaustiveWord = "exhaustive"

// lowerExhaustiveClauses emits the checked Go default for each exhaustive clause.
func lowerExhaustiveClauses(
	files *token.FileSet,
	file *token.File,
	tree *syntax.File,
	data []byte,
	edits []edit,
) ([]edit, map[[2]int]bool, error) {
	var failure error
	locations := make(map[[2]int]bool)
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
			position := files.Position(clause.Exhaustive)
			locations[[2]int{position.Line, position.Column}] = true
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
			edits = append(edits,
				edit{start: start, end: start + len(exhaustiveWord), text: "default"},
				edit{
					start: file.Offset(clause.Colon) + 1,
					end:   file.Offset(clause.Colon) + 1,
					text: " panic(" + string(data[receiverStart:receiverEnd]) +
						".UnknownTag()) " + enumDefaultComment,
				},
			)
		}
		return true
	})
	return edits, locations, failure
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
