package navigation_test

import (
	"bufio"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tgo/internal/navigation"
)

type fixtureExpectation struct {
	File          string `json:"file"`
	Use           string `json:"use"`
	UseOccurrence int    `json:"useOccurrence"`
	Definition    string `json:"definition"`
}

func TestHelperDefinitionProtocol(t *testing.T) {
	repository := repositoryRoot(t)
	helper := filepath.Join(t.TempDir(), "tgonav")
	command := exec.Command("go", "build", "-o", helper, "./cmd/tgonav")
	command.Dir = repository
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, output)
	}

	workspace := filepath.Join(
		repository, "internal", "navigation", "testdata", "workspaces", "basic",
	)
	expectation := readExpectation(t, filepath.Join(workspace, "expect.json"))
	sourcePath := filepath.Join(workspace, expectation.File)
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	use := nthOffset(t, string(source), expectation.Use, expectation.UseOccurrence)
	definition := strings.Index(string(source), expectation.Definition)
	if definition < 0 {
		t.Fatalf("definition %q is absent", expectation.Definition)
	}

	process := exec.Command(helper, "-root", workspace)
	input, err := process.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := process.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	process.Stderr = os.Stderr
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}

	encoder := json.NewEncoder(input)
	decoder := json.NewDecoder(bufio.NewReader(output))
	request := navigation.Request{
		ID:     1,
		Method: "definition",
		Params: mustJSON(t, map[string]any{
			"uri":    fileURI(t, sourcePath),
			"offset": use,
		}),
	}
	if err := encoder.Encode(request); err != nil {
		t.Fatal(err)
	}
	var response struct {
		ID     int64                 `json:"id"`
		Result []navigation.Location `json:"result"`
		Error  string                `json:"error"`
	}
	if err := decoder.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "" {
		t.Fatalf("helper error: %s", response.Error)
	}
	if response.ID != 1 {
		t.Fatalf("response ID = %d, want 1", response.ID)
	}
	want := navigation.Location{
		URI:   fileURI(t, sourcePath),
		Start: definition,
		End:   definition + len(expectation.Definition),
	}
	if len(response.Result) != 1 || response.Result[0] != want {
		t.Fatalf("definition = %#v, want %#v", response.Result, want)
	}
	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	if err := process.Wait(); err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("find test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readExpectation(t *testing.T, path string) fixtureExpectation {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result fixtureExpectation
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func nthOffset(t *testing.T, source, needle string, occurrence int) int {
	t.Helper()
	offset := 0
	for current := 1; current <= occurrence; current++ {
		next := strings.Index(source[offset:], needle)
		if next < 0 {
			t.Fatalf("occurrence %d of %q is absent", occurrence, needle)
		}
		offset += next
		if current == occurrence {
			return offset
		}
		offset += len(needle)
	}
	return -1
}

func fileURI(t *testing.T, path string) string {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	value := new(url.URL)
	value.Scheme = "file"
	value.Path = filepath.ToSlash(absolute)
	return value.String()
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
