package compilerv2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeReturnChecks(t *testing.T, dir string, site Site) {
	t.Helper()
	reference, err := os.ReadFile("testdata/proof/reference.go")
	if err != nil {
		t.Fatal(err)
	}
	reference = []byte(strings.TrimPrefix(string(reference), "//go:build proofreference\n"))
	if err := os.WriteFile(filepath.Join(dir, "reference.go"), reference, 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/proof/expect.json")
	if err != nil {
		t.Fatal(err)
	}
	var inputs []proofInput
	if err := json.Unmarshal(data, &inputs); err != nil {
		t.Fatal(err)
	}
	var checks strings.Builder
	checks.WriteString(`package proof
import("testing";"reflect";"errors")
var proofVisits,proofExit int
func TestReturn(t *testing.T){sentinel:=errors.New("sentinel")
`)
	for _, input := range inputs {
		for visit := 1; visit <= 2; visit++ {
			fmt.Fprintf(&checks, `t.Run(%q,func(t *testing.T){Trace=nil;Reset(%t);proofVisits=0;proofExit=%d
var result error;panicked:=false
func(){defer func(){if recover()!=nil{panicked=true}}();result=Case%s(sentinel)}()
expected,returned,wantPanic:=ReferenceReturn(%q,%d,%d,%q,%t,%d)
if !reflect.DeepEqual(Trace,expected){t.Fatalf("trace: %%v, want %%v",Trace,expected)}
if panicked!=wantPanic{t.Fatalf("panic: %%v, want %%v",panicked,wantPanic)}
if returned {if result!=sentinel {t.Fatalf("result: %%v, want sentinel identity",result)}} else if result!=nil {t.Fatalf("result: %%v, want nil",result)}
})
`, fmt.Sprintf("%s/%t/%d", input.Name, input.Reached, visit), input.Reached, visit, input.Name, filepath.Base(site.File), site.Start, site.End, input.Name, input.Reached, visit)
		}
	}
	checks.WriteString("}\n")
	if err := os.WriteFile(filepath.Join(dir, "proof_test.go"), []byte(checks.String()), 0600); err != nil {
		t.Fatal(err)
	}
}
