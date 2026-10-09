package format

import (
	"go/token"

	"tgo/pkg/syntax"
)

func (p *printer) binaryExpressionAt(
	value *syntax.BinaryExpression,
	parentPrecedence int,
	depth int,
) {
	precedence := value.Operator.Precedence()
	parenthesize := precedence < parentPrecedence
	if parenthesize {
		p.text("(")
	}
	p.binaryOperand(value.Left, precedence, depth+binaryDepthChange(value.Left, precedence))
	spaces := precedence < binaryCutoff(value, depth)
	if spaces {
		p.space()
	}
	p.token(value.OperatorPosition, value.Operator.String())
	operatorEnd := p.tokenEnd(value.OperatorPosition, len(value.Operator.String()))
	p.trailingToken(value.OperatorPosition, len(value.Operator.String()))
	if p.multiline(syntax.ExpressionEnd(value.Left), syntax.ExpressionPosition(value.Right)) {
		p.indent++
		p.breakSourceGap(operatorEnd, syntax.ExpressionPosition(value.Right))
		p.binaryOperand(value.Right, precedence+1, depth+1)
		p.indent--
	} else {
		if spaces {
			p.space()
		}
		p.binaryOperand(value.Right, precedence+1, depth+1)
	}
	if parenthesize {
		p.text(")")
	}
}

func (p *printer) binaryOperand(value *syntax.Expression, precedence int, depth int) {
	if binary := syntax.BinaryExpressionOf(value); binary != nil {
		p.binaryExpressionAt(binary, precedence, depth)
		return
	}
	p.expressionAt(value, precedence, depth)
}

func binaryDepthChange(value *syntax.Expression, precedence int) int {
	binary := syntax.BinaryExpressionOf(value)
	if binary != nil && binary.Operator.Precedence() == precedence {
		return 0
	}
	return 1
}

func binaryCutoff(value *syntax.BinaryExpression, depth int) int {
	has4, has5, problem := binaryShape(value)
	if problem > 0 {
		return problem + 1
	}
	if has4 && has5 {
		if depth == 1 {
			return 5
		}
		return 4
	}
	if depth == 1 {
		return 6
	}
	return 4
}

func binaryShape(value *syntax.BinaryExpression) (bool, bool, int) {
	has4 := value.Operator.Precedence() == 4
	has5 := value.Operator.Precedence() == 5
	problem := 0
	if left := syntax.BinaryExpressionOf(value.Left); left != nil &&
		left.Operator.Precedence() >= value.Operator.Precedence() {
		left4, left5, leftProblem := binaryShape(left)
		has4 = has4 || left4
		has5 = has5 || left5
		problem = max(problem, leftProblem)
	}
	if right := syntax.BinaryExpressionOf(value.Right); right != nil &&
		right.Operator.Precedence() > value.Operator.Precedence() {
		right4, right5, rightProblem := binaryShape(right)
		has4 = has4 || right4
		has5 = has5 || right5
		problem = max(problem, rightProblem)
	} else if syntax.StarExpressionOf(value.Right) != nil && value.Operator == token.QUO {
		problem = 5
	} else if unary := syntax.UnaryExpressionOf(value.Right); unary != nil {
		switch value.Operator.String() + unary.Operator.String() {
		case "/*", "&&", "&^":
			problem = 5
		case "++", "--":
			problem = max(problem, 4)
		}
	}
	return has4, has5, problem
}
