package frontend

import (
	"fmt"
	"go/token"
	"strings"
	"testing"
)

func BenchmarkConditionalParsing(b *testing.B) {
	for _, count := range []int{1, 32, 128, 512} {
		for _, nested := range []bool{false, true} {
			name, source := conditionalBenchmarkSource(count, nested)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(source),
						make(map[string]bool)); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func conditionalBenchmarkSource(count int, nested bool) (string, string) {
	name := fmt.Sprintf("independent/%d", count)
	var body strings.Builder
	if nested {
		name = fmt.Sprintf("nested/%d", count)
		for i := 0; i < count; i++ {
			body.WriteString("if ok { (")
		}
		body.WriteString("1")
		for i := 0; i < count; i++ {
			body.WriteString(") } else { 2 }")
		}
	} else {
		for i := 0; i < count; i++ {
			body.WriteString("_ = if ok { 1 } else { 2 };\n")
		}
	}
	source := "package p\nfunc f(ok bool) {\n" + body.String() + "\n}"
	if nested {
		source = "package p\nfunc f(ok bool) int { return " + body.String() + "\n}"
	}
	return name, source
}
