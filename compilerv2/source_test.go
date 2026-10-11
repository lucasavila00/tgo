package compilerv2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSourceReturnTargets(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/input\n\ngo 1.27.1\n",
		"input.go": `package input
var global = 1
func run(err error) error {
 const fixed = 2
 _ = fixed
 _ = func() int { return 3 }
 _ = func() error { return err }
 for range (*[2]int)(nil) {}
 defer run(err)
 go run(err)
 return err
}
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, err := Load(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	text := files["input.go"]
	found := map[string]bool{}
	for _, site := range source.Sites {
		expression := text[site.Start:site.End]
		switch expression {
		case "(*[2]int)(nil)":
			if site.Reason != "range expression is not evaluated" {
				t.Fatalf("range conversion: %+v", site)
			}
			found["range conversion"] = true
		case "1":
			if site.Reason != "package scope" {
				t.Fatalf("global: %+v", site)
			}
			found["global"] = true
		case "2":
			if site.Reason == "range expression is not evaluated" {
				break
			}
			if site.Reason != "constant declaration" {
				t.Fatalf("constant: %+v", site)
			}
			found["constant"] = true
		case "3":
			if site.ErrorReturn {
				t.Fatalf("closure has integer return: %+v", site)
			}
			found["integer closure"] = true
		case "err":
			if site.Reason == "" && !site.ErrorReturn {
				t.Fatalf("error return target: %+v", site)
			}
		case "run(err)":
			if site.Reason == "call executes in another goroutine" {
				found["go"] = true
			}
			if site.Reason == "call executes when the function returns" {
				found["defer"] = true
			}
		}
	}
	for _, name := range []string{"global", "constant", "integer closure", "go", "defer", "range conversion"} {
		if !found[name] {
			t.Errorf("missing %s", name)
		}
	}
}
