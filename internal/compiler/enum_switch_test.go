package compiler

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"testing"
)

func TestEnumCaseTagsRejectsRepeatedTags(t *testing.T) {
	first := ast.NewIdent("first")
	second := ast.NewIdent("second")
	p := &packageUnit{fs: token.NewFileSet(), info: newInfo()}
	p.info.Types[first] = types.TypeAndValue{Value: constant.MakeInt64(1)}
	p.info.Types[second] = types.TypeAndValue{Value: constant.MakeInt64(1)}
	clause := &ast.CaseClause{List: []ast.Expr{first, second}}

	tags := p.enumCaseTags(clause, 2, map[int]bool{})

	if !tags[1] || len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	if len(p.errors) != 1 {
		t.Fatalf("errors: %v", p.errors)
	}
}

func TestEnumCaseTagsRejectsTagFromEarlierCase(t *testing.T) {
	expression := ast.NewIdent("repeated")
	p := &packageUnit{fs: token.NewFileSet(), info: newInfo()}
	p.info.Types[expression] = types.TypeAndValue{Value: constant.MakeInt64(1)}
	clause := &ast.CaseClause{List: []ast.Expr{expression}}

	tags := p.enumCaseTags(clause, 2, map[int]bool{1: true})

	if len(tags) != 0 || len(p.errors) != 1 {
		t.Fatalf("tags=%v errors=%v", tags, p.errors)
	}
}
