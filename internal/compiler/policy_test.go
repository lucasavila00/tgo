package compiler

import (
	"go/importer"
	"go/token"
	"testing"
)

func TestCompilerLeavesModelUsagePolicyToTgolint(t *testing.T) {
	t.Parallel()
	compiled, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Account enum {
	Personal struct { Name string }
}

type AccountLayout struct {
	tgoTag AccountTag
	tgoPersonal AccountPersonal
}

type Direct Account
type Accounts interface { Account }
type Embedded struct { Account }

var literal = Account{}

func construct(layout AccountLayout) Account { return Account(layout) }
func expose(value Account) AccountLayout { return AccountLayout(value) }
func exposePointer(value *Account) *AccountLayout { return (*AccountLayout)(value) }
func constructGeneric[T Accounts](layout AccountLayout) T { return T(layout) }
func exposeGeneric[T Accounts](value T) AccountLayout { return AccountLayout(value) }
func promoted(value Embedded) AccountTag { return value.tgoTag }

func anonymous(value Account) AccountPersonal {
	var exposed struct {
		tgoTag AccountTag
		tgoPersonal AccountPersonal
	} = value
	return exposed.tgoPersonal
}
`)}},
		FileSet:  token.NewFileSet(),
		Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compiler applied source usage policy: %v", problems)
	}
	if compiled == nil || len(compiled.Outputs["sample.tgo"]) == 0 {
		t.Fatal("compiler did not emit source for tgolint")
	}
}
