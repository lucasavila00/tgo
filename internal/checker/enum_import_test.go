package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"golang.org/x/tools/go/packages"
)

// TestEnumImportsKeepSourceIdentity checks an enum returned through a compiled Go wrapper.
func TestEnumImportsKeepSourceIdentity(t *testing.T) {
	for _, wrapperFirst := range []bool{true, false} {
		name := "enum-first"
		imports := "import accounts \"example.com/external/accounts\"\nimport \"example.com/external/wrapper\"\n"
		if wrapperFirst {
			name = "wrapper-first"
			imports = "import \"example.com/external/wrapper\"\nimport accounts \"example.com/external/accounts\"\n"
		}
		t.Run(name, func(t *testing.T) {
			root, external := enumImportModule(t, imports, "type Alias = accounts.Account\nvar Value = wrapper.Get()\n")
			before := readImportSources(t, external)
			loader := loadEnumImportModule(t, root)
			checkImportedEnumIdentity(t, loader)
			after := readImportSources(t, external)
			for name, content := range before {
				if after[name] != content {
					t.Errorf("dependency source changed: %s", name)
				}
			}
		})
	}
}

// TestEnumImportsWithoutSource keeps the ordinary generated Go API usable without native metadata.
func TestEnumImportsWithoutSource(t *testing.T) {
	root, external := enumImportModule(t,
		"import accounts \"example.com/external/accounts\"\nimport \"example.com/external/wrapper\"\n",
		"var Value = wrapper.Get()\nvar Other = accounts.NewAccountBusiness(\"company\")\n")
	if err := os.Remove(filepath.Join(external, "accounts", "account.tgo")); err != nil {
		t.Fatal(err)
	}
	loader := loadEnumImportModule(t, root)
	if loader.packages["example.com/external/accounts"] != nil {
		t.Fatal("compiled-only package was checked from source")
	}
	account := loader.imports["example.com/external/accounts"].Scope().Lookup("Account").Type()
	if types.EnumOf(account) != nil {
		t.Fatal("ordinary Go API was mistaken for a native enum declaration")
	}
}

// TestEnumImportedMatches checks cases that name imported variants through aliases and case-only imports.
func TestEnumImportedMatches(t *testing.T) {
	for name, body := range map[string]string{
		"case-only": `func Read() string {
	switch value := wrapper.Get().(type) {
	case accounts.Account.Personal: return value.Name
	case accounts.Account.Business: return value.Company
	exhaustive:
	}
}`,
		"alias": `type Alias = accounts.Account
func Read() string {
	switch value := wrapper.Get().(type) {
	case Alias.Personal: return value.Name
	case Alias.Business: return value.Company
	exhaustive:
	}
}`,
	} {
		t.Run(name, func(t *testing.T) {
			root, _ := enumImportModule(t,
				"import \"example.com/external/wrapper\"\nimport accounts \"example.com/external/accounts\"\n", body)
			checkImportedEnumIdentity(t, loadEnumImportModule(t, root))
		})
	}
}

// loadEnumImportModule selects Go dependencies and checks the staged TGo root with one loader.
func loadEnumImportModule(t *testing.T, root string) *Loader {
	t.Helper()
	roots, err := packages.Load(&packages.Config{Dir: root, Mode: packages.NeedName | packages.NeedFiles |
		packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedExportFile |
		packages.NeedModule}, ".")
	if err != nil || len(roots) != 1 {
		t.Fatalf("load packages: %v, %v", roots, err)
	}
	sizes, err := TargetSizes(root)
	if err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(root, roots, map[string]string{"main_tgo.go": "main.tgo"}, sizes)
	if _, err := loader.Check(roots[0]); err != nil {
		t.Fatal(err)
	}
	return loader
}

// enumImportModule writes native enum input, its Go API, and an ordinary Go wrapper in another module.
func enumImportModule(t *testing.T, imports, body string) (string, string) {
	t.Helper()
	base := t.TempDir()
	root, external := filepath.Join(base, "root"), filepath.Join(base, "external")
	files := map[string]string{
		"root/go.mod": "module example.com/root\n\ngo 1.27.1\n" +
			"require example.com/external v0.0.0\nreplace example.com/external => ../external\n",
		"external/go.mod": "module example.com/external\n\ngo 1.27.1\n",
		"external/accounts/account.tgo": `package accounts

type Account enum {
	Personal struct { Name string }
	Business struct { Company string }
}
`,
		"external/accounts/account_tgo.go": enumImportGoAPI,
		"external/wrapper/wrapper.go": `package wrapper
import "example.com/external/accounts"
type Alias = accounts.Account
func Get() Alias { return accounts.NewAccountPersonal("name") }
`,
		"root/main_tgo.go": "package root\n" + imports + body,
	}
	for name, source := range files {
		path := filepath.Join(base, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root, external
}

const enumImportGoAPI = `package accounts

type Account struct { tag AccountTag; personal AccountPersonal; business AccountBusiness }
type AccountTag uint8
const (AccountTagPersonal AccountTag = iota; AccountTagBusiness)
type AccountPersonal struct { Name string }
type AccountBusiness struct { Company string }
func NewAccountPersonal(name string) Account { return Account{personal: AccountPersonal{name}} }
func NewAccountBusiness(company string) Account { return Account{tag: 1, business: AccountBusiness{company}} }
func (account Account) Tag() AccountTag { return account.tag }
func (account Account) PersonalPayload() AccountPersonal { return account.personal }
func (account Account) BusinessPayload() AccountBusiness { return account.business }
`

// checkImportedEnumIdentity follows a compiled function result back to the original enum declaration.
func checkImportedEnumIdentity(t *testing.T, loader *Loader) {
	t.Helper()
	dependency := loader.packages["example.com/external/accounts"]
	if dependency == nil {
		t.Fatal("enum dependency was not checked from source")
	}
	named := dependency.Types.Scope().Lookup("Account").Type()
	enum := types.EnumOf(named)
	if enum == nil {
		t.Fatal("source enum metadata is missing")
	}
	wrapper := loader.imports["example.com/external/wrapper"]
	result := wrapper.Scope().Lookup("Get").Type().(*types.Signature).Results().At(0).Type()
	if types.Unalias(result) != named {
		t.Fatal("compiled wrapper has a different enum type identity")
	}
	specification := dependency.Syntax[0].Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	if enum.Declaration() != specification.Type || dependency.TypesInfo.Defs[specification.Name].Type() != named {
		t.Fatal("enum metadata lost its source node or definition identity")
	}
}

// readImportSources records dependency bytes so source loading can be checked for writes.
func readImportSources(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		if err == nil {
			files[strings.TrimPrefix(path, root)] = string(content)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// TestEnumImportsUseSelectedFiles reads only the dependency inputs selected by Go build metadata.
func TestEnumImportsUseSelectedFiles(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(root, "external")
	if err := os.Mkdir(external, 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"main_tgo.go": "package p; import \"example.com/accounts\"; " +
			"var Value = accounts.NewAccountFirst()",
		"external/account_linux_amd64.tgo":    "package accounts; type Account enum { First struct {} }",
		"external/account_tgo_linux_amd64.go": "package accounts; type Account struct{}",
		"external/helper.go":                  "package accounts; var Other Account",
		"external/unselected.tgo":             "this file must not be parsed",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	dependency := &packages.Package{ID: "example.com/accounts", PkgPath: "example.com/accounts",
		GoFiles: []string{filepath.Join(external, "account_tgo_linux_amd64.go"), filepath.Join(external, "helper.go")}}
	metadata := &packages.Package{ID: "example.com/p", PkgPath: "example.com/p",
		GoFiles: []string{filepath.Join(root, "main_tgo.go")},
		Imports: map[string]*packages.Package{dependency.PkgPath: dependency}}
	loader := NewLoader(root, []*packages.Package{metadata}, map[string]string{"main_tgo.go": "main.tgo"},
		types.SizesFor("gc", "amd64"))
	if _, err := loader.Check(metadata); err != nil {
		t.Fatal(err)
	}
	checked := loader.packages[dependency.ID]
	if len(checked.Syntax) != 2 || types.EnumOf(checked.Types.Scope().Lookup("Account").Type()) == nil {
		t.Fatal("selected TGo input and ordinary Go file were not checked together")
	}
}

// TestEnumImportsInTestVariants checks original sources and separate identities in dependency test variants.
func TestEnumImportsInTestVariants(t *testing.T) {
	root := enumImportTestModule(t)
	roots, testPackage, ordinary := loadEnumImportTestVariants(t, root)
	variant := testPackage.Imports[ordinary.PkgPath]
	sizes, err := TargetSizes(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, order := range []string{"variant-first", "ordinary-first"} {
		t.Run(order, func(t *testing.T) {
			loader := NewLoader(root, roots, map[string]string{
				"main_tgo.go": "main.tgo", "external_tgo_test.go": "external_test.tgo",
			}, sizes)
			if order == "ordinary-first" {
				if err := loader.prepareSourceImports(ordinary); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := loader.Check(testPackage); err != nil {
				t.Fatal(err)
			}
			if order == "variant-first" && loader.imports[ordinary.PkgPath] != nil {
				t.Fatal("test variant replaced the ordinary import")
			}
			if err := loader.prepareSourceImports(ordinary); err != nil {
				t.Fatal(err)
			}
			checkEnumImportTestIdentities(t, loader, ordinary, variant)
		})
	}
}

// enumImportTestModule adds a test that imports a dependency test variant.
func enumImportTestModule(t *testing.T) string {
	t.Helper()
	root, external := enumImportModule(t, "", "type Root struct{}\n")
	for name, source := range map[string]string{
		filepath.Join(root, "main_test.go"): "package root\nvar TestValue Root\n",
		filepath.Join(external, "accounts", "root.go"): "package accounts\n" +
			"import \"example.com/root\"\nvar RootValue root.Root\n",
		filepath.Join(root, "external_tgo_test.go"): `package root_test
import "example.com/external/accounts"
import "example.com/external/wrapper"
func Read() string {
	switch value := wrapper.Get().(type) {
	case accounts.Account.Personal: return value.Name
	case accounts.Account.Business: return value.Company
	exhaustive:
	}
}
`,
	} {
		if err := os.WriteFile(name, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// loadEnumImportTestVariants selects the ordinary dependency and its test variant.
func loadEnumImportTestVariants(t *testing.T, root string) ([]*packages.Package, *packages.Package, *packages.Package) {
	t.Helper()
	roots, err := packages.Load(&packages.Config{Dir: root, Tests: true, Mode: packages.NeedName |
		packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps |
		packages.NeedExportFile | packages.NeedModule}, ".", "example.com/external/accounts")
	if err != nil {
		t.Fatal(err)
	}
	var testPackage, ordinary *packages.Package
	for _, metadata := range roots {
		switch metadata.PkgPath {
		case "example.com/root_test":
			testPackage = metadata
		case "example.com/external/accounts":
			if metadata.ID == metadata.PkgPath {
				ordinary = metadata
			}
		}
	}
	if testPackage == nil || ordinary == nil {
		t.Fatalf("missing test or ordinary package: %v", roots)
	}
	variant := testPackage.Imports[ordinary.PkgPath]
	if variant == nil || variant.ID == ordinary.ID {
		t.Fatal("Go did not select a dependency test variant")
	}
	return roots, testPackage, ordinary
}

// checkEnumImportTestIdentities checks both enum identities and the ordinary import.
func checkEnumImportTestIdentities(t *testing.T, loader *Loader, ordinary, variant *packages.Package) {
	t.Helper()
	first, second := loader.packages[ordinary.ID], loader.packages[variant.ID]
	if first == nil || second == nil || first.Types == second.Types {
		t.Fatal("ordinary package and test variant do not have separate identities")
	}
	if loader.imports[ordinary.PkgPath] != first.Types {
		t.Fatal("ordinary import does not use the ordinary package identity")
	}
	for _, pkg := range []*Package{first, second} {
		if types.EnumOf(pkg.Types.Scope().Lookup("Account").Type()) == nil {
			t.Fatal("enum metadata is missing from a package identity")
		}
	}
}
