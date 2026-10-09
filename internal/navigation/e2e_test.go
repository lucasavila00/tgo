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

type fixturePoint struct {
	File       string `json:"file"`
	Text       string `json:"text"`
	Occurrence int    `json:"occurrence"`
}

type fixtureRequest struct {
	Method             string         `json:"method"`
	Position           fixturePoint   `json:"position"`
	IncludeDeclaration bool           `json:"includeDeclaration"`
	Locations          []fixturePoint `json:"locations"`
}

type helperProcess struct {
	command *exec.Cmd
	input   *json.Encoder
	output  *json.Decoder
	nextID  int64
}

func TestHelperWorkspaceFixtures(t *testing.T) {
	repository := repositoryRoot(t)
	helper := buildHelper(t, repository)
	workspaces := filepath.Join(
		repository, "internal", "navigation", "testdata", "workspaces",
	)
	entries, err := os.ReadDir(workspaces)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			workspace := filepath.Join(workspaces, entry.Name())
			server := startHelper(t, helper, workspace)
			for _, request := range readRequests(t, workspace) {
				server.check(t, workspace, request)
			}
		})
	}
}

func buildHelper(t *testing.T, repository string) string {
	t.Helper()
	helper := filepath.Join(t.TempDir(), "tgonav")
	command := exec.Command("go", "build", "-o", helper, "./cmd/tgonav")
	command.Dir = repository
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, output)
	}
	return helper
}

func startHelper(t *testing.T, helper, workspace string) *helperProcess {
	t.Helper()
	command := exec.Command(helper, "-root", workspace)
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = input.Close()
		if err := command.Wait(); err != nil {
			t.Errorf("stop helper: %v", err)
		}
	})
	return &helperProcess{
		command: command,
		input:   json.NewEncoder(input),
		output:  json.NewDecoder(bufio.NewReader(output)),
		nextID:  1,
	}
}

func (h *helperProcess) check(
	t *testing.T,
	workspace string,
	fixture fixtureRequest,
) {
	t.Helper()
	positionPath := filepath.Join(workspace, fixture.Position.File)
	offset := pointOffset(t, positionPath, fixture.Position)
	request := navigation.Request{
		ID:     h.nextID,
		Method: fixture.Method,
		Params: mustJSON(t, map[string]any{
			"uri":                fileURI(t, positionPath),
			"offset":             offset,
			"includeDeclaration": fixture.IncludeDeclaration,
		}),
	}
	if err := h.input.Encode(request); err != nil {
		t.Fatal(err)
	}
	var response struct {
		ID     int64                 `json:"id"`
		Result []navigation.Location `json:"result"`
		Error  string                `json:"error"`
	}
	if err := h.output.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "" {
		t.Fatalf("helper error: %s", response.Error)
	}
	if response.ID != h.nextID {
		t.Fatalf("response ID = %d, want %d", response.ID, h.nextID)
	}
	h.nextID++
	want := make([]navigation.Location, 0, len(fixture.Locations))
	for _, point := range fixture.Locations {
		path := filepath.Join(workspace, point.File)
		start := pointOffset(t, path, point)
		want = append(want, navigation.Location{
			URI: fileURI(t, path), Start: start, End: start + len(point.Text),
		})
	}
	if !equalLocations(response.Result, want) {
		t.Fatalf(
			"%s at %#v result = %#v, want %#v",
			fixture.Method, fixture.Position, response.Result, want,
		)
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

func readRequests(t *testing.T, workspace string) []fixtureRequest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(workspace, "requests.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result []fixtureRequest
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func pointOffset(t *testing.T, path string, point fixturePoint) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return nthOffset(t, string(data), point.Text, point.Occurrence)
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

func equalLocations(left, right []navigation.Location) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
