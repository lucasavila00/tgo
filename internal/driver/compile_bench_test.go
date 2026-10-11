package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkCompilePropagations measures a complete compile with warm Go caches.
func BenchmarkCompilePropagations(b *testing.B) {
	for _, count := range []int{1, 8, 32} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			dir := b.TempDir()
			source := "package p\nfunc call() (int, error) { return 1, nil }\nfunc run() error {\n" + strings.Repeat("_ = call()!\n", count) + "return nil\n}\n"
			for name, content := range map[string]string{"go.mod": "module example.com/bench\n\ngo 1.27.1\n", "main.tgo": source} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
					b.Fatal(err)
				}
			}
			Compile(dir)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				Compile(dir)
			}
		})
	}
}
