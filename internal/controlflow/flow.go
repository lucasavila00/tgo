package controlflow

import (
	"go/ast"
	"go/token"
)

// Syntax identifies statements inside added expressions and Go control-flow rules.
type Syntax struct {
	RangeSourceOutside func(*ast.RangeStmt) bool
	Statements         func(*ast.CallExpr) ast.Stmt
	Terminates         func(*ast.CallExpr) bool
	Unevaluated        func(*ast.CallExpr) bool
}

// Result reports whether a branch can reach its last statement or its end.
type Result struct {
	Tail bool
	End  bool
}

type point struct{ next []*point }
type targets struct{ stop, next, fall *point }
type graph struct {
	syntax    Syntax
	labels    map[string]*point
	jumps     map[string]targets
	tail      ast.Stmt
	tailPoint *point
}

// Branch follows all source paths through a branch. Jumps out of the branch end that path.
func Branch(body *ast.BlockStmt, syntax Syntax) Result {
	g := &graph{syntax: syntax, labels: make(map[string]*point), jumps: make(map[string]targets)}
	list := body.List
	for len(list) > 0 {
		if _, empty := list[len(list)-1].(*ast.EmptyStmt); !empty {
			break
		}
		list = list[:len(list)-1]
	}
	if len(list) > 0 {
		g.tail = list[len(list)-1]
	}
	done := &point{}
	entry := g.statements(body.List, done, targets{})
	seen := make(map[*point]bool)
	pending := []*point{entry}
	for len(pending) > 0 {
		n := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if n == nil || seen[n] {
			continue
		}
		seen[n] = true
		pending = append(pending, n.next...)
	}
	return Result{Tail: seen[g.tailPoint], End: seen[done]}
}

func link(next ...*point) *point { return &point{next: next} }
func (g *graph) label(name string) *point {
	if g.labels[name] == nil {
		g.labels[name] = &point{}
	}
	return g.labels[name]
}
func (g *graph) statements(list []ast.Stmt, next *point, t targets) *point {
	for i := len(list) - 1; i >= 0; i-- {
		next = g.statement(list[i], next, t, "")
	}
	return next
}
func (g *graph) statement(s ast.Stmt, next *point, t targets, label string) *point {
	entry := g.statementInner(s, next, t, label)
	if s == g.tail {
		if _, labelled := s.(*ast.LabeledStmt); labelled {
			g.tailPoint = entry
			return entry
		}
		g.tailPoint = link(entry)
		return g.tailPoint
	}
	return entry
}
func (g *graph) statementInner(s ast.Stmt, next *point, t targets, label string) *point {
	if s == nil {
		return next
	}
	switch s := s.(type) {
	case *ast.BlockStmt:
		return g.statements(s.List, next, t)
	case *ast.LabeledStmt:
		entry := g.label(s.Label.Name)
		entry.next = []*point{g.statement(s.Stmt, next, t, s.Label.Name)}
		return entry
	case *ast.BranchStmt:
		switch s.Tok {
		case token.GOTO:
			return g.label(s.Label.Name)
		case token.BREAK:
			if s.Label != nil {
				return link(g.jumps[s.Label.Name].stop)
			}
			return link(t.stop)
		case token.CONTINUE:
			if s.Label != nil {
				return link(g.jumps[s.Label.Name].next)
			}
			return link(t.next)
		case token.FALLTHROUGH:
			return link(t.fall)
		}
	case *ast.ReturnStmt:
		return g.expressions(s.Results, link(), t)
	case *ast.ExprStmt:
		return g.expression(s.X, next, t)
	case *ast.AssignStmt:
		return g.expressions(append(append([]ast.Expr{}, s.Lhs...), s.Rhs...), next, t)
	case *ast.DeclStmt:
		if decl, ok := s.Decl.(*ast.GenDecl); ok {
			for i := len(decl.Specs) - 1; i >= 0; i-- {
				if spec, ok := decl.Specs[i].(*ast.ValueSpec); ok {
					next = g.expressions(spec.Values, next, t)
				}
			}
		}
		return next
	case *ast.SendStmt:
		return g.expressions([]ast.Expr{s.Chan, s.Value}, next, t)
	case *ast.IncDecStmt:
		return g.expression(s.X, next, t)
	case *ast.GoStmt:
		return g.callOperands(s.Call, next, t)
	case *ast.DeferStmt:
		return g.callOperands(s.Call, next, t)
	case *ast.IfStmt:
		choice := link(g.statement(s.Body, next, t, ""), g.statement(s.Else, next, t, ""))
		return g.statement(s.Init, g.expression(s.Cond, choice, t), t, "")
	case *ast.ForStmt:
		return g.forStatement(s, next, t, label)
	case *ast.RangeStmt:
		return g.rangeStatement(s, next, t, label)
	case *ast.SwitchStmt:
		return g.switchStatement(s, next, t, label)
	case *ast.TypeSwitchStmt:
		return g.typeSwitch(s, next, t, label)
	case *ast.SelectStmt:
		return g.selectStatement(s, next, t, label)
	}
	return next
}

func (g *graph) expression(expr ast.Expr, next *point, t targets) *point {
	if expr == nil {
		return next
	}
	switch e := expr.(type) {
	case *ast.CallExpr:
		if g.syntax.Unevaluated != nil && g.syntax.Unevaluated(e) {
			return next
		}
		if g.syntax.Statements != nil {
			if body := g.syntax.Statements(e); body != nil {
				return g.statement(body, next, t, "")
			}
		}
		if g.syntax.Terminates != nil && g.syntax.Terminates(e) {
			next = link()
		}
		return g.callOperands(e, next, t)
	case *ast.BinaryExpr:
		right := g.expression(e.Y, next, t)
		if e.Op == token.LAND || e.Op == token.LOR {
			right = link(next, right)
		}
		return g.expression(e.X, right, t)
	case *ast.UnaryExpr:
		return g.expression(e.X, next, t)
	case *ast.ParenExpr:
		return g.expression(e.X, next, t)
	case *ast.SelectorExpr:
		return g.expression(e.X, next, t)
	case *ast.IndexExpr:
		return g.expressions([]ast.Expr{e.X, e.Index}, next, t)
	case *ast.IndexListExpr:
		return g.expression(e.X, next, t)
	case *ast.SliceExpr:
		return g.expressions([]ast.Expr{e.X, e.Low, e.High, e.Max}, next, t)
	case *ast.StarExpr:
		return g.expression(e.X, next, t)
	case *ast.TypeAssertExpr:
		return g.expression(e.X, next, t)
	case *ast.CompositeLit:
		return g.expressions(e.Elts, next, t)
	case *ast.KeyValueExpr:
		return g.expressions([]ast.Expr{e.Key, e.Value}, next, t)
	}
	return next
}
func (g *graph) expressions(exprs []ast.Expr, next *point, t targets) *point {
	for i := len(exprs) - 1; i >= 0; i-- {
		next = g.expression(exprs[i], next, t)
	}
	return next
}
func (g *graph) callOperands(call *ast.CallExpr, next *point, t targets) *point {
	return g.expression(call.Fun, g.expressions(call.Args, next, t), t)
}
func (g *graph) forStatement(s *ast.ForStmt, next *point, t targets, label string) *point {
	cond := &point{}
	post := &point{}
	inner := targets{stop: next, next: post, fall: t.fall}
	if label != "" {
		g.jumps[label] = inner
		defer delete(g.jumps, label)
	}
	post.next = []*point{g.statement(s.Post, cond, inner, "")}
	body := g.statement(s.Body, post, inner, "")
	choice := body
	if s.Cond != nil {
		choice = link(body, next)
	}
	cond.next = []*point{g.expression(s.Cond, choice, inner)}
	return g.statement(s.Init, cond, inner, "")
}
func (g *graph) rangeStatement(s *ast.RangeStmt, next *point, t targets, label string) *point {
	step := &point{}
	inner := targets{stop: next, next: step, fall: t.fall}
	if label != "" {
		g.jumps[label] = inner
		defer delete(g.jumps, label)
	}
	body := g.statement(s.Body, step, inner, "")
	body = g.expressions([]ast.Expr{s.Key, s.Value}, body, inner)
	step.next = []*point{body, next}
	if g.syntax.RangeSourceOutside != nil && g.syntax.RangeSourceOutside(s) {
		inner = t
	}
	return g.expression(s.X, step, inner)
}
func (g *graph) switchStatement(s *ast.SwitchStmt, next *point, t targets, label string) *point {
	inner := targets{stop: next, next: t.next}
	if label != "" {
		g.jumps[label] = inner
		defer delete(g.jumps, label)
	}
	bodies := make([]*point, len(s.Body.List))
	fall := next
	defaultBody := next
	for i := len(bodies) - 1; i >= 0; i-- {
		clause := s.Body.List[i].(*ast.CaseClause)
		inner.fall = fall
		bodies[i] = g.statements(clause.Body, next, inner)
		fall = bodies[i]
		if clause.List == nil {
			defaultBody = bodies[i]
		}
	}
	choice := defaultBody
	for i := len(bodies) - 1; i >= 0; i-- {
		clause := s.Body.List[i].(*ast.CaseClause)
		for j := len(clause.List) - 1; j >= 0; j-- {
			choice = g.expression(clause.List[j], link(bodies[i], choice), inner)
		}
	}
	return g.statement(s.Init, g.expression(s.Tag, choice, inner), inner, "")
}
func (g *graph) typeSwitch(s *ast.TypeSwitchStmt, next *point, t targets, label string) *point {
	inner := targets{stop: next, next: t.next}
	if label != "" {
		g.jumps[label] = inner
		defer delete(g.jumps, label)
	}
	choice := &point{}
	hasDefault := false
	for _, stmt := range s.Body.List {
		clause := stmt.(*ast.CaseClause)
		hasDefault = hasDefault || clause.List == nil
		choice.next = append(choice.next, g.statements(clause.Body, next, inner))
	}
	if !hasDefault {
		choice.next = append(choice.next, next)
	}
	return g.statement(s.Init, g.statement(s.Assign, choice, inner, ""), inner, "")
}
func (g *graph) selectStatement(s *ast.SelectStmt, next *point, t targets, label string) *point {
	inner := targets{stop: next, next: t.next}
	if label != "" {
		g.jumps[label] = inner
		defer delete(g.jumps, label)
	}
	choice := &point{}
	var operands []ast.Expr
	for _, stmt := range s.Body.List {
		clause := stmt.(*ast.CommClause)
		body := g.statements(clause.Body, next, inner)
		switch comm := clause.Comm.(type) {
		case *ast.SendStmt:
			operands = append(operands, comm.Chan, comm.Value)
		case *ast.ExprStmt:
			operands = append(operands, ast.Unparen(comm.X).(*ast.UnaryExpr).X)
		case *ast.AssignStmt:
			operands = append(operands, ast.Unparen(comm.Rhs[0]).(*ast.UnaryExpr).X)
			body = g.expressions(comm.Lhs, body, inner)
		}
		choice.next = append(choice.next, body)
	}
	return g.expressions(operands, choice, inner)
}
