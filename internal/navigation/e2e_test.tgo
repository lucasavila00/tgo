package navigation_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
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
	Hover              string         `json:"hover"`
}

type fixtureSymbol struct {
	Name      string       `json:"name"`
	Kind      string       `json:"kind"`
	Container string       `json:"container"`
	Range     fixturePoint `json:"range"`
	Selection fixturePoint `json:"selection"`
}

type fixtureSymbolRequest struct {
	Method  string          `json:"method"`
	File    string          `json:"file"`
	Query   string          `json:"query"`
	Names   []string        `json:"names"`
	Symbols []fixtureSymbol `json:"symbols"`
}

type protocolSymbol struct {
	Name      string              `json:"name"`
	Kind      string              `json:"kind"`
	Container string              `json:"container"`
	Range     navigation.Location `json:"range"`
	Selection navigation.Location `json:"selection"`
}

type fixtureMutation struct {
	File    string         `json:"file"`
	Old     string         `json:"old"`
	New     string         `json:"new"`
	Request fixtureRequest `json:"request"`
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
			workspace := filepath.Join(t.TempDir(), entry.Name())
			copyWorkspace(t, filepath.Join(workspaces, entry.Name()), workspace)
			server := startHelper(t, helper, workspace)
			for _, request := range readRequests(t, workspace) {
				server.check(t, workspace, request)
			}
			for _, request := range readSymbolRequests(t, workspace) {
				server.checkSymbols(t, workspace, request)
			}
			for _, mutation := range readMutations(t, workspace) {
				server.checkMutation(t, workspace, mutation)
			}
		})
	}
}

func TestHelperCancellationStopsBeforeInvalidPackage(t *testing.T) {
	repository := repositoryRoot(t)
	helper := buildHelper(t, repository)
	source := filepath.Join(
		repository, "internal", "navigation", "testdata", "workspaces", "cancellation",
	)
	workspace := filepath.Join(t.TempDir(), "cancellation")
	copyWorkspace(t, source, workspace)
	server := startHelper(t, helper, workspace)
	if err := server.input.Encode(map[string]any{
		"id": 1, "method": "workspaceSymbols",
		"params": map[string]any{"query": ""},
	}); err != nil {
		t.Fatal(err)
	}
	if err := server.input.Encode(map[string]any{
		"method": "cancel", "params": map[string]any{"id": 1},
	}); err != nil {
		t.Fatal(err)
	}
	response := struct {
		ID    int64  `json:"id"`
		Error string `json:"error"`
	}{ID: 0, Error: ""}
	if err := server.output.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.ID != 1 || response.Error != context.Canceled.Error() {
		t.Fatalf("canceled response = %#v", response)
	}
}

func (h *helperProcess) checkMutation(
	t *testing.T,
	workspace string,
	fixture fixtureMutation,
) {
	t.Helper()
	path := filepath.Join(workspace, fixture.File)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), fixture.Old) == 0 {
		t.Fatalf("mutation text %q is absent from %s", fixture.Old, fixture.File)
	}
	changed := strings.ReplaceAll(string(data), fixture.Old, fixture.New)
	if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidated := false
	h.call(t, "invalidate", mustJSON(t, map[string]any{
		"uri": fileURI(t, path),
	}), &invalidated)
	if !invalidated {
		t.Fatal("helper did not confirm invalidation")
	}
	h.check(t, workspace, fixture.Request)
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
	params := mustJSON(t, map[string]any{
		"uri":                fileURI(t, positionPath),
		"offset":             offset,
		"includeDeclaration": fixture.IncludeDeclaration,
	})
	if fixture.Method == "hover" {
		result := (*navigation.Hover)(nil)
		h.call(t, fixture.Method, params, &result)
		want := &navigation.Hover{
			Contents: fixture.Hover,
			Range: navigation.Location{
				URI: fileURI(t, positionPath), Start: offset,
				End: offset + len(fixture.Position.Text),
			},
		}
		if result == nil || *result != *want {
			t.Fatalf(
				"hover at %#v result = %#v, want %#v",
				fixture.Position, result, want,
			)
		}
		return
	}
	result := []navigation.Location(nil)
	h.call(t, fixture.Method, params, &result)
	want := make([]navigation.Location, 0, len(fixture.Locations))
	for _, point := range fixture.Locations {
		path := filepath.Join(workspace, point.File)
		start := pointOffset(t, path, point)
		want = append(want, navigation.Location{
			URI: fileURI(t, path), Start: start, End: start + len(point.Text),
		})
	}
	if !equalLocations(result, want) {
		t.Fatalf(
			"%s at %#v result = %#v, want %#v",
			fixture.Method, fixture.Position, result, want,
		)
	}
}

func (h *helperProcess) checkSymbols(
	t *testing.T,
	workspace string,
	fixture fixtureSymbolRequest,
) {
	t.Helper()
	params := map[string]any{"query": fixture.Query}
	if fixture.File != "" {
		params["uri"] = fileURI(t, filepath.Join(workspace, fixture.File))
	}
	result := []protocolSymbol(nil)
	h.call(t, fixture.Method, mustJSON(t, params), &result)
	names := make([]string, 0, len(result))
	for _, symbol := range result {
		names = append(names, symbol.Name)
	}
	if !equalStrings(names, fixture.Names) {
		t.Fatalf("%s names = %#v, want %#v", fixture.Method, names, fixture.Names)
	}
	byName := make(map[string]protocolSymbol)
	for _, symbol := range result {
		byName[symbol.Name] = symbol
	}
	for _, expected := range fixture.Symbols {
		actual, ok := byName[expected.Name]
		if !ok {
			t.Fatalf("symbol %q is absent", expected.Name)
		}
		rangePath := filepath.Join(workspace, expected.Range.File)
		rangeStart := pointOffset(t, rangePath, expected.Range)
		selectionPath := filepath.Join(workspace, expected.Selection.File)
		selectionStart := pointOffset(t, selectionPath, expected.Selection)
		want := protocolSymbol{
			Name: expected.Name, Kind: expected.Kind, Container: expected.Container,
			Range: navigation.Location{
				URI: fileURI(t, rangePath), Start: rangeStart,
				End: rangeStart + len(expected.Range.Text),
			},
			Selection: navigation.Location{
				URI: fileURI(t, selectionPath), Start: selectionStart,
				End: selectionStart + len(expected.Selection.Text),
			},
		}
		if actual != want {
			t.Fatalf("symbol %q = %#v, want %#v", expected.Name, actual, want)
		}
	}
}

func (h *helperProcess) call(
	t *testing.T,
	method string,
	params json.RawMessage,
	result any,
) {
	t.Helper()
	request := map[string]any{
		"id": h.nextID, "method": method, "params": params,
	}
	if err := h.input.Encode(request); err != nil {
		t.Fatal(err)
	}
	var response struct {
		ID     int64           `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	} = struct {
		ID     int64           `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}{ID: 0, Result: nil, Error: ""}
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
	if err := json.Unmarshal(response.Result, result); err != nil {
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

func readRequests(t *testing.T, workspace string) []fixtureRequest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(workspace, "requests.json"))
	if err != nil {
		t.Fatal(err)
	}
	result := []fixtureRequest(nil)
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func readSymbolRequests(t *testing.T, workspace string) []fixtureSymbolRequest {
	t.Helper()
	path := filepath.Join(workspace, "symbols.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	result := []fixtureSymbolRequest(nil)
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func readMutations(t *testing.T, workspace string) []fixtureMutation {
	t.Helper()
	path := filepath.Join(workspace, "invalidation.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	result := []fixtureMutation(nil)
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func copyWorkspace(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
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

func equalStrings(left, right []string) bool {
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
