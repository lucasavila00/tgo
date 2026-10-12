package driver

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkCompileAssets(b *testing.B) {
	for _, size := range []int{0, 32 << 20} {
		name := "none"
		if size != 0 {
			name = "32MiB"
		}
		b.Run(name, func(b *testing.B) {
			root := b.TempDir()
			files := map[string][]byte{"go.mod": []byte("module example.com/assets\n\ngo 1.27.1\n"),
				"main.tgo": []byte("package p\nfunc call()(int,error){return 1,nil}\nfunc run()error{_ = if true " +
					"{call()!}else{0};return nil}\n"),
				"asset.bin": bytes.Repeat([]byte{42}, size)}
			for name, content := range files {
				if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
					b.Fatal(err)
				}
			}
			Compile(root)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				Compile(root)
			}
		})
	}
}
