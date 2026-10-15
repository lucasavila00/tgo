package format

import (
	"bytes"
	stock "go/format"
	"go/token"
	"testing"

	"github.com/lucasavila00/tgo/internal/frontend/parser"
)

// TestPrivateFormatMatchesGo checks ordinary source and node formatting against the stock formatter.
func TestPrivateFormatMatchesGo(t *testing.T) {
	for _, source := range []string{
		"package p; import(\"strings\";\"fmt\");func f(x string)string{return fmt.Sprint(strings.TrimSpace(x))}",
		"package p\n// kept\nfunc identity[T any](v T) T{return v}\nvar values=map[string]int{\"one\":1}",
		"package p; func f(x bool)bool{return !x}",
	} {
		t.Run(source, func(t *testing.T) {
			want, err := stock.Source([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
			got, err := Source([]byte(source))
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("source output: %s; error: %v; want: %s", got, err, want)
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "main.go", source, parser.ParseComments|parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := Node(&output, set, file); err != nil || !bytes.Equal(output.Bytes(), want) {
				t.Fatalf("node output: %s; error: %v; want: %s", output.Bytes(), err, want)
			}
		})
	}
}
