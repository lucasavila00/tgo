package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkCompileComprehensions(b *testing.B) {
	for _, count := range []int{1, 16, 64} {
		for _, nested := range []bool{false, true} {
			b.Run(fmt.Sprintf("count=%d/nested=%t", count, nested), func(b *testing.B) {
				benchmarkComprehensionCase(b, count, nested)
			})
		}
	}
}

func benchmarkComprehensionCase(b *testing.B, count int, nested bool) {
	root := b.TempDir()
	var body strings.Builder
	body.WriteString("package p; func f(xs []int) { ")
	if nested {
		expression := "xs"
		for range count {
			expression = "[]int{x for _, x := range " + expression + "}"
		}
		body.WriteString("_ = " + expression)
	} else {
		for range count {
			body.WriteString("_ = []int{x for _, x := range xs};")
		}
	}
	body.WriteString(" }")
	for name, content := range map[string]string{
		"go.mod":   "module example.com/bench\n\ngo 1.27.1\n",
		"main.tgo": body.String(),
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}
	Compile(root)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Compile(root)
	}
}
