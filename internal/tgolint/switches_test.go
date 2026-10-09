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
	tagType := testTagType()
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{
		first:  {Value: constant.MakeInt64(1)},
		second: {Value: constant.MakeInt64(1)},
	}, Uses: map[*ast.Ident]types.Object{
		first:  types.NewConst(0, nil, "AccountTagPersonal", tagType, constant.MakeInt64(1)),
		second: types.NewConst(0, nil, "AccountTagPersonal", tagType, constant.MakeInt64(1)),
	}}
	reports := 0
	c := &checker{pass: &analysis.Pass{
		TypesInfo: info,
		Report:    func(analysis.Diagnostic) { reports++ },
	}}
	clause := &ast.CaseClause{List: []ast.Expr{first, second}}

	tags, _ := c.caseTags(clause, testTagModel(), tagType, map[int]bool{})

	if !tags[1] || len(tags) != 1 || reports != 1 {
		t.Fatalf("tags=%v reports=%d", tags, reports)
	}
}

func TestCaseTagsRejectsTagFromEarlierCase(t *testing.T) {
	expression := ast.NewIdent("repeated")
	tagType := testTagType()
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{
		expression: {Value: constant.MakeInt64(1)},
	}, Uses: map[*ast.Ident]types.Object{
		expression: types.NewConst(0, nil, "AccountTagPersonal", tagType, constant.MakeInt64(1)),
	}}
	reports := 0
	c := &checker{pass: &analysis.Pass{
		TypesInfo: info,
		Report:    func(analysis.Diagnostic) { reports++ },
	}}
	clause := &ast.CaseClause{List: []ast.Expr{expression}}

	tags, _ := c.caseTags(clause, testTagModel(), tagType, map[int]bool{1: true})

	if len(tags) != 0 || reports != 1 {
		t.Fatalf("tags=%v reports=%d", tags, reports)
	}
}

func testTagType() types.Type {
	return types.NewNamed(types.NewTypeName(0, nil, "AccountTag", nil), types.Typ[types.Uint8], nil)
}

func testTagModel() *model {
	return enumModel("", "Account", []string{"Personal", "Business"})
}
