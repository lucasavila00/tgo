package compiler

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestExhaustiveClauseLowersToCheckedDefault(t *testing.T) {
	p := layoutPackage(t, `package sample
type Account enum { Personal struct{}; Business struct{} }
func use(account Account) {
	switch account.Tag() {
	case AccountTagPersonal:
		return
	case AccountTagBusiness:
		return
	exhaustive:
	}
}
`)
	if len(p.Sources[0].Exhaustive) != 1 {
		t.Fatalf("exhaustive clauses = %d", len(p.Sources[0].Exhaustive))
	}
	var clause *ast.CaseClause
	ast.Inspect(p.Sources[0].File, func(node ast.Node) bool {
		item, ok := node.(*ast.CaseClause)
		if ok && len(item.List) == 0 {
			clause = item
		}
		return true
	})
	if clause == nil || len(clause.Body) != 1 {
		t.Fatal("exhaustive clause did not emit a default panic")
	}
}

func TestEnumDefaultAllowsFallback(t *testing.T) {
	layoutPackage(t, `package sample
type Account enum {
	Personal struct { Name string }
	Business struct { Company string }
}
func use(account Account) string {
	switch account.Tag() {
	case AccountTagPersonal:
		return account.PersonalPayload().Name
	default:
		return account.BusinessPayload().Company
	}
}
`)
}

func TestExhaustiveClauseRejectsBody(t *testing.T) {
	_, err := parseSource(token.NewFileSet(), "sample.tgo", []byte(`package sample
type Account enum { Personal struct{} }
func use(account Account) {
	switch account.Tag() {
	case AccountTagPersonal:
		return
	exhaustive:
		return
	}
}
`))
	if err == nil || !strings.Contains(err.Error(), "exhaustive clause must not have a body") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumCaseTagsRejectsRepeatedTags(t *testing.T) {
	first := ast.NewIdent("first")
	second := ast.NewIdent("second")
	p := &packageUnit{fs: token.NewFileSet(), info: newInfo()}
	tagType := testEnumTagType()
	p.info.Types[first] = types.TypeAndValue{Value: constant.MakeInt64(1)}
	p.info.Types[second] = types.TypeAndValue{Value: constant.MakeInt64(1)}
	p.info.Uses[first] = types.NewConst(
		0, nil, "AccountTagPersonal", tagType, constant.MakeInt64(1),
	)
	p.info.Uses[second] = types.NewConst(
		0, nil, "AccountTagPersonal", tagType, constant.MakeInt64(1),
	)
	clause := &ast.CaseClause{List: []ast.Expr{first, second}}

	tags, _ := p.enumCaseTags(clause, testEnumModel(), tagType, map[int]bool{})

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
	tagType := testEnumTagType()
	p.info.Types[expression] = types.TypeAndValue{Value: constant.MakeInt64(1)}
	p.info.Uses[expression] = types.NewConst(
		0, nil, "AccountTagPersonal", tagType, constant.MakeInt64(1),
	)
	clause := &ast.CaseClause{List: []ast.Expr{expression}}

	tags, _ := p.enumCaseTags(clause, testEnumModel(), tagType, map[int]bool{1: true})

	if len(tags) != 0 || len(p.errors) != 1 {
		t.Fatalf("tags=%v errors=%v", tags, p.errors)
	}
}

func testEnumTagType() types.Type {
	return types.NewNamed(types.NewTypeName(0, nil, "AccountTag", nil), types.Typ[types.Uint8], nil)
}

func testEnumModel() *model {
	return &model{Name: "Account", Variants: []variant{{Name: "Personal"}, {Name: "Business"}}}
}
