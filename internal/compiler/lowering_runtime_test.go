package compiler

import (
	"context"
	"go/importer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

type loweringRuntimeFixture struct {
	path        string
	sourceName  string
	source      string
	testName    string
	testSource  string
	testPattern string
	extraFiles  map[string][]byte
	validate    func(*testing.T, *CompiledPackage)
}

func runLoweringRuntimeTest(t *testing.T, fixture loweringRuntimeFixture) {
	t.Helper()
	compiled, problems := Compile(PackageInput{
		Path: fixture.path,
		Sources: []File{
			{Name: fixture.sourceName, Data: []byte(fixture.source)},
			{Name: fixture.testName, Data: []byte(fixture.testSource)},
		},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}
	if fixture.validate != nil {
		fixture.validate(t, compiled)
	}

	files := map[string][]byte{
		"go.mod":                         []byte("module " + fixture.path + "\n\ngo 1.27.0\n"),
		goSourceName(fixture.sourceName): compiled.Outputs[fixture.sourceName],
		goSourceName(fixture.testName):   compiled.Outputs[fixture.testName],
	}
	for name, data := range fixture.extraFiles {
		files[name] = data
	}
	directory := t.TempDir()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx, "go", "test", "-run", fixture.testPattern, "-count=1", ".",
	)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

func goSourceName(name string) string {
	return name[:len(name)-len(".tgo")] + ".go"
}
