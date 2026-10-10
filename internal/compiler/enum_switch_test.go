package compiler

import (
	"go/ast"
	"go/importer"
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

func TestExhaustiveClauseStoresCallReceiverOnce(t *testing.T) {
	output := compileEnumSwitch(t, `package sample
type Event enum { Ready struct{} }
func use(load func() Event) string {
	switch load().Tag() {
	case EventTagReady:
		return "ready"
	exhaustive:
	}
}
`)
	if strings.Count(output, ":= load()") != 1 {
		t.Fatalf("receiver evaluation count is not one\n%s", output)
	}
	for _, text := range []string{
		"switch enumValue := load(); enumValue.Tag()",
		`panic("invalid Event tag")`,
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated switch does not contain %q\n%s", text, output)
		}
	}
}

func TestExhaustiveClauseStoresReceiverAfterInitializer(t *testing.T) {
	output := compileEnumSwitch(t, `package sample
type Event enum { Ready struct{} }
func use(load func() Event, enumValue Event) string {
	switch initialized := true; // evaluate the initializer first
	(load()).Tag() {
	case EventTagReady:
		_ = initialized
		return "ready"
		exhaustive:
	}
}
`)
	for _, text := range []string{
		"initialized := true // evaluate the initializer first\n" +
			"\t\tenumValue_1 := (load())\n\t\tswitch enumValue_1.Tag()",
		"// evaluate the initializer first",
		`panic("invalid Event tag")`,
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated switch does not contain %q\n%s", text, output)
		}
	}
}

func TestExhaustiveClauseKeepsLocalReceivers(t *testing.T) {
	output := compileEnumSwitch(t, `package sample
type Event enum { Ready struct{} }
func valueReceiver(value Event) {
	switch (value).Tag() {
	case EventTagReady:
		return
	exhaustive:
	}
}
func pointerReceiver(value *Event) {
	switch value.Tag() {
	case EventTagReady:
		return
	exhaustive:
	}
}
`)
	if strings.Contains(output, "enumValue :=") {
		t.Fatalf("local receivers gained redundant storage\n%s", output)
	}
	for _, text := range []string{
		"switch value.Tag()", `panic("invalid Event tag")`,
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated switch does not contain %q\n%s", text, output)
		}
	}
}

func TestExhaustiveClauseStoresIndexReceiverOnce(t *testing.T) {
	output := compileEnumSwitch(t, `package sample
type Event enum { Ready struct{} }
func use(values []Event, index func() int) {
	switch values[index()].Tag() {
	case EventTagReady:
		return
	exhaustive:
	}
}
`)
	if strings.Count(output, "values[index()]") != 1 {
		t.Fatalf("index receiver evaluation count is not one\n%s", output)
	}
	for _, text := range []string{
		"switch enumValue := values[index()]; enumValue.Tag()",
		`panic("invalid Event tag")`,
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated switch does not contain %q\n%s", text, output)
		}
	}
}

func TestExhaustiveClauseNamesLiteralReceiverEnum(t *testing.T) {
	output := compileEnumSwitch(t, `package sample
type Event enum { Ready struct{} }
func use() {
	switch (Event.Ready{}).Tag() {
	case EventTagReady:
		return
	exhaustive:
	}
}
`)
	if !strings.Contains(output, `panic("invalid Event tag")`) {
		t.Fatalf("generated switch does not name Event\n%s", output)
	}
}

func TestExhaustiveClausePreservesLabeledGoto(t *testing.T) {
	output := compileEnumSwitch(t, `package sample
type Event enum { Ready struct{} }
func use(load func() Event, again bool) string {
	goto dispatch
dispatch:
	switch initialized := true; load().Tag() {
	case EventTagReady:
		_ = initialized
		if again {
			again = false
			goto dispatch
		}
		break dispatch
	exhaustive:
	}
	return "done"
}
`)
	for _, text := range []string{
		"dispatch:\n\tswitch {\n\tdefault:\n\t\tinitialized := true",
		"enumValue := load()\n\t\tswitch enumValue.Tag()",
		"goto dispatch", "break dispatch",
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated switch does not contain %q\n%s", text, output)
		}
	}
}

func compileEnumSwitch(t *testing.T, source string) string {
	t.Helper()
	compiled, problems := Compile(PackageInput{
		Path:    "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(source)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile errors: %v", problems)
	}
	return string(compiled.Outputs["sample.tgo"])
}
