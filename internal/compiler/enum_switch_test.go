package compiler

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

func TestExhaustiveClauseLowersToGoDefault(t *testing.T) {
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
