package lowering

import (
	"fmt"
	"strings"
	"testing"
)

// TestEnumTagNames keeps the generated tag's integer type when source names hide builtins.
func TestEnumTagNames(t *testing.T) {
	cases := map[string]string{
		"canonical": `package main
import "fmt"
type E enum { A; B }
func main() { fmt.Println(E.B{}.Tag() == ETagB) }
`,
		"byte alias": `package main
import "fmt"
type uint8 string
type E enum { A; B }
func main() { fmt.Println(E.B{}.Tag() == ETagB) }
`,
		"import alias": `package main
import uint8 "fmt"
type E enum { A; B }
func main() { uint8.Println(E.B{}.Tag() == ETagB) }
`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			checkNativeOutput(t, input, "true\n")
		})
	}
}

// TestEnumTagNamesUnavailable reports the source enum before invalid Go can be emitted.
func TestEnumTagNamesUnavailable(t *testing.T) {
	var large strings.Builder
	large.WriteString("package main\ntype uint16 string\ntype E enum {\n")
	for index := range 257 {
		fmt.Fprintf(&large, "V%d\n", index)
	}
	large.WriteString("}\nfunc main() {}\n")
	cases := map[string]struct {
		input    string
		typeName string
	}{
		"both byte names": {"package main\ntype uint8 string\ntype byte string\ntype E enum { A; B }\n", "uint8"},
		"uint16":          {large.String(), "uint16"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			source, _ := loadNativeSource(t, test.input)
			_, err := source.LowerPropagations()
			want := "cannot name type " + test.typeName + " for enum E tag"
			if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "main.tgo:") {
				t.Fatalf("got %v, want a source diagnostic containing %q", err, want)
			}
		})
	}
}
