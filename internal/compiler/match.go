package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

func (p *packageUnit) lowerMatches() {
	for len(p.errors) == 0 {
		changed := false
		for _, source := range p.Sources {
			astutil.Apply(source.File, func(cursor *astutil.Cursor) bool {
				if changed {
					return false
				}
				statement, ok := cursor.Node().(*ast.SwitchStmt)
				if !ok {
					return true
				}
				tag, ok := statement.Tag.(*ast.CallExpr)
				if !ok || !ident(tag.Fun, "__tgo_match") {
					return true
				}
				cursor.Replace(p.lowerMatch(statement, tag))
				changed = true
				return false
			}, nil)
		}
		if !changed {
			return
		}
		// An outer case introduces bindings needed to type nested matches.
		if err := p.typecheck(false); err != nil {
			p.errors = append(p.errors, err)
			return
		}
	}
}

func (p *packageUnit) lowerMatch(statement *ast.SwitchStmt, tag *ast.CallExpr) ast.Node {
	if len(tag.Args) != 1 {
		p.fail(statement, "match needs one value")
		return statement
	}
	model := p.modelForType(p.info.TypeOf(tag.Args[0]))
	if model == nil || len(model.Variants) == 0 {
		p.fail(statement, "match needs an enum value")
		return statement
	}
	p.serial++
	temporary := fmt.Sprintf("__tgo_match_%d", p.serial)
	seen := make(map[string]bool)
	for _, body := range statement.Body.List {
		clause := body.(*ast.CaseClause)
		p.lowerCase(clause, model, temporary, seen)
	}
	for _, variant := range model.Variants {
		if !seen[variant.Name] {
			p.fail(statement, "match is missing variant %s", variant.Name)
		}
	}
	statement.Tag = methodCall(temporary, "TgoTag")
	statement.Body.List = append(statement.Body.List, invalidVariantCase(model.Name))
	binding := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent(temporary)},
		Tok: token.DEFINE,
		Rhs: tag.Args,
	}
	return &ast.BlockStmt{List: []ast.Stmt{binding, statement}}
}

func (p *packageUnit) lowerCase(
	clause *ast.CaseClause,
	model *model,
	temporary string,
	seen map[string]bool,
) {
	name, binding, err := casePattern(clause)
	if err != nil {
		p.fail(clause, "%s", err)
		return
	}
	index := -1
	for position, variant := range model.Variants {
		if variant.Name == name {
			index = position
		}
	}
	if index < 0 {
		p.fail(clause, "unknown variant %s", name)
		return
	}
	if seen[name] {
		p.fail(clause, "duplicate variant %s", name)
	}
	seen[name] = true
	clause.List = []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(index + 1)}}
	if binding.Name == "_" {
		return
	}
	assignment := &ast.AssignStmt{
		Lhs: []ast.Expr{binding},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{methodCall(temporary, "Tgo"+name)},
	}
	clause.Body = append([]ast.Stmt{assignment}, clause.Body...)
}

func casePattern(clause *ast.CaseClause) (string, *ast.Ident, error) {
	if len(clause.List) != 1 {
		return "", nil, fmt.Errorf("match needs one variant per case")
	}
	pattern, ok := clause.List[0].(*ast.CallExpr)
	if !ok || len(pattern.Args) != 1 {
		return "", nil, fmt.Errorf("use case Variant(binding)")
	}
	name, ok := pattern.Fun.(*ast.Ident)
	if !ok {
		return "", nil, fmt.Errorf("invalid variant name")
	}
	binding, ok := pattern.Args[0].(*ast.Ident)
	if !ok {
		return "", nil, fmt.Errorf("variant binding must be a name")
	}
	return name.Name, binding, nil
}

func methodCall(receiver, method string) *ast.CallExpr {
	return call(&ast.SelectorExpr{X: ast.NewIdent(receiver), Sel: ast.NewIdent(method)})
}

func invalidVariantCase(name string) *ast.CaseClause {
	message := &ast.BasicLit{
		Kind:  token.STRING,
		Value: strconv.Quote("invalid " + name + " variant"),
	}
	failure := &ast.ExprStmt{X: call(ast.NewIdent("panic"), message)}
	return &ast.CaseClause{Body: []ast.Stmt{failure}}
}
