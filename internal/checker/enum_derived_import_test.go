package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnumImportedDefinedType keeps ordinary Go conversions through a source Go wrapper.
func TestEnumImportedDefinedType(t *testing.T) {
	imports := []string{
		"import accounts \"example.com/external/accounts\"\nimport \"example.com/external/wrapper\"\n",
		"import \"example.com/external/wrapper\"\nimport accounts \"example.com/external/accounts\"\n",
	}
	for _, imports := range imports {
		t.Run(imports, func(t *testing.T) {
			root, external := enumImportModule(t, imports+"import \"example.com/external/facade\"\n", `
var Value = accounts.Account(wrapper.Other(accounts.Account{}))
var Returned = accounts.Account(wrapper.Get())
var Aliased = accounts.Account(wrapper.Alias{})
var ThroughFacade = accounts.Account(facade.Get())
`)
			writeDefinedEnumWrapper(t, external)
			loader := loadEnumImportModule(t, root)
			if loader.packages["example.com/external/wrapper"] == nil ||
				loader.packages["example.com/external/facade"] == nil {
				t.Fatal("enum wrapper must be checked from source to preserve defined-type identity")
			}
		})
	}
}

// writeDefinedEnumWrapper supplies an ordinary Go definition with the enum's exact generated shape.
func writeDefinedEnumWrapper(t *testing.T, external string) {
	t.Helper()
	api := strings.NewReplacer("tag", "tgoTag", "personal", "tgoVariant0", "business", "tgoVariant1").
		Replace(enumImportGoAPI)
	wrapper := `package wrapper
import "example.com/external/accounts"
type Other accounts.Account
type Alias = Other
func Get() Other { return Other(accounts.NewAccountPersonal("name")) }
`
	facade := `package facade
import "example.com/external/wrapper"
type Third wrapper.Other
func Get() Third { return Third(wrapper.Get()) }
`
	for name, source := range map[string]string{
		"accounts/account_tgo.go": api, "wrapper/wrapper.go": wrapper, "facade/facade.go": facade,
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(external, name)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(external, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

// TestOrdinaryTGoDependencyKeepsCompiledWrapper avoids source checks when no enum metadata is needed.
func TestOrdinaryTGoDependencyKeepsCompiledWrapper(t *testing.T) {
	root, external := enumImportModule(t,
		"import accounts \"example.com/external/accounts\"\nimport \"example.com/external/wrapper\"\n",
		"type Alias = accounts.Account; var Value = wrapper.Get()")
	if err := os.WriteFile(filepath.Join(external, "accounts", "account.tgo"), []byte(enumImportGoAPI), 0600); err != nil {
		t.Fatal(err)
	}
	loader := loadEnumImportModule(t, root)
	if loader.packages["example.com/external/accounts"] == nil {
		t.Fatal("TGo dependency source was not checked")
	}
	if loader.packages["example.com/external/wrapper"] != nil {
		t.Fatal("ordinary Go wrapper did not need a source check")
	}
}
