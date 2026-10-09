package tgolint

import (
	"go/ast"
	"go/constant"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestCaseTagsRejectsRepeatedTags(t *testing.T) {
	first := ast.NewIdent("first")
	second := ast.NewIdent("second")
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{
		first:  {Value: constant.MakeInt64(1)},
		second: {Value: constant.MakeInt64(1)},
	}}
	reports := 0
	c := checker{pass: &analysis.Pass{
		TypesInfo: info,
		Report:    func(analysis.Diagnostic) { reports++ },
	}}
	clause := &ast.CaseClause{List: []ast.Expr{first, second}}

	tags := c.caseTags(clause, 2, map[int]bool{})

	if !tags[1] || len(tags) != 1 || reports != 1 {
		t.Fatalf("tags=%v reports=%d", tags, reports)
	}
}

func TestCaseTagsRejectsTagFromEarlierCase(t *testing.T) {
	expression := ast.NewIdent("repeated")
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{
		expression: {Value: constant.MakeInt64(1)},
	}}
	reports := 0
	c := checker{pass: &analysis.Pass{
		TypesInfo: info,
		Report:    func(analysis.Diagnostic) { reports++ },
	}}
	clause := &ast.CaseClause{List: []ast.Expr{expression}}

	tags := c.caseTags(clause, 2, map[int]bool{1: true})

	if len(tags) != 0 || reports != 1 {
		t.Fatalf("tags=%v reports=%d", tags, reports)
	}
}
