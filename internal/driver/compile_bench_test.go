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
	benchmarkPropagations(b, false)
}

func BenchmarkCompileNestedPropagations(b *testing.B) {
	benchmarkPropagations(b, true)
}

func benchmarkPropagations(b *testing.B, nested bool) {
	for _, count := range []int{1, 8, 32} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			dir := b.TempDir()
			calls := strings.Repeat("_ = call()!\n", count)
			if nested {
				expression := "call()!"
				for i := 1; i < count; i++ {
					expression = "outer(" + expression + ")!"
				}
				calls = "_ = " + expression + "\n"
			}
			source := "package p\nfunc call() (int, error) { return 1, nil }\nfunc outer(v int) " +
				"(int,error) {return v,nil}\nfunc run() error {\n" +
				calls + "return nil\n}\n"
			for name, content := range map[string]string{"go.mod": "module example.com/bench\n\ngo 1.27.1\n",
				"main.tgo": source} {
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

func BenchmarkCompilePackagePropagations(b *testing.B) {
	dir := b.TempDir()
	files := map[string]string{"go.mod": "module example.com/bench\n\ngo 1.27.1\n"}
	calls := strings.Repeat("_ = call()!\n", 8)
	files["dep/dep.tgo"] = "package dep\nfunc call() (int,error) {return 1,nil}\nfunc Run() error {\n" + calls +
		"return nil\n}\n"
	files["main.tgo"] = "package p\nimport \"example.com/bench/dep\"\nfunc call() (int,error) {return 1," +
		"nil}\nfunc Run() error {dep.Run()!\n" +
		calls + "return nil\n}\n"
	files["main_test.tgo"] = "package p\nimport \"testing\"\nfunc check() error {\n" + calls +
		"return nil\n}\nfunc TestRun(t *testing.T) {if err:=check();err!=nil " +
		"{t.Fatal(err)}}\n"
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			b.Fatal(err)
		}
	}
	Compile(dir)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Compile(dir)
	}
}
