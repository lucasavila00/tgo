package compiler

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tgo/internal/outputname"
)

func TestOrdinaryGoIdentity(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/ordinary-go")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		if !fixture.IsDir() {
			continue
		}
		t.Run(fixture.Name(), func(t *testing.T) {
			sources := readIdentityFixture(t, filepath.Join("testdata/ordinary-go", fixture.Name()))
			checkOrdinaryGoIdentity(t, sources)
		})
	}
}

func TestGeneratedOrdinaryGoIdentity(t *testing.T) {
	const seed int64 = 0x54474f
	random := rand.New(rand.NewSource(seed))
	sources := make(map[string][]byte)
	for index := range 64 {
		name := fmt.Sprintf("generated_%02d.tgo", index)
		sources[name] = []byte(generatedGoSource(index, random.Intn(4), random.Intn(97)+1))
	}
	t.Logf("seed: %d", seed)
	checkOrdinaryGoIdentity(t, sources)
}

func TestVerifyOrdinaryGoIdentity(t *testing.T) {
	source := []byte("package sample\n\ntype Pair struct{}\n")
	if err := VerifyGeneratedModels("sample.tgo", source, source, nil); err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(source, []byte("struct{}"), []byte("struct{ Value int }"), 1)
	if err := VerifyGeneratedModels("sample.tgo", source, changed, nil); err == nil {
		t.Fatal("ownership verification accepted changed ordinary Go")
	}
}

func generatedGoSource(index, shape, value int) string {
	switch shape {
	case 0:
		return fmt.Sprintf(`package sample

type Pair%d struct{ Left, Right int }

func Sum%d(pair Pair%d) int { return pair.Left + pair.Right + %d }
`, index, index, index, value)
	case 1:
		return fmt.Sprintf(`package sample

func Select%d(value int) int {
	if value < %d { return value }
	switch value %% 2 { case 0: return value / 2; default: return value * 3 + 1 }
}
`, index, value)
	case 2:
		return fmt.Sprintf(`package sample

type Number%d interface { ~int | ~int64 }

func Add%d[T Number%d](values []T) T {
	total := T(%d)
	for _, value := range values { total += value }
	return total
}
`, index, index, index, value)
	default:
		return fmt.Sprintf(`package sample

func Map%d(values []int) map[int]int {
	result := map[int]int{}
	for index, value := range values { result[index] = value + %d }
	return result
}
`, index, value)
	}
}

func readIdentityFixture(t *testing.T, fixture string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(fixture)
	if err != nil {
		t.Fatal(err)
	}
	sources := make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fixture, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(entry.Name(), ".input")
		sources[name] = data
	}
	return sources
}

func checkOrdinaryGoIdentity(t *testing.T, sources map[string][]byte) {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module identity.test\n\ngo 1.27.0\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(sources))
	for name, data := range sources {
		names = append(names, name)
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := Build(directory, nil); err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	for _, name := range names {
		want := sources[name]
		output := outputname.Path(name)
		got, err := os.ReadFile(filepath.Join(directory, output))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf(
				"%s changed ordinary Go\n%s\nsource:\n%s",
				name,
				lineDiff(want, got),
				sources[name],
			)
		}
	}
}

func lineDiff(want, got []byte) string {
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(string(got), "\n")
	limit := min(len(wantLines), len(gotLines))
	for index := 0; index < limit; index++ {
		if wantLines[index] != gotLines[index] {
			return fmt.Sprintf(
				"line %d:\nwant %q\ngot  %q",
				index+1,
				wantLines[index],
				gotLines[index],
			)
		}
	}
	return fmt.Sprintf("want %d lines, got %d lines", len(wantLines), len(gotLines))
}
