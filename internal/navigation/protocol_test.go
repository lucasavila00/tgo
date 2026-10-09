package navigation

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRequestJSONVariants(t *testing.T) {
	tests := []struct {
		name string
		wire string
		tag  RequestTag
	}{
		{
			"hover",
			`{"id":1,"method":"hover","params":{"uri":"file:///a.tgo","offset":4}}`,
			RequestTagHover,
		},
		{
			"definition",
			`{"id":2,"method":"definition","params":{"uri":"file:///a.tgo","offset":5}}`,
			RequestTagDefinition,
		},
		{
			"references",
			`{"id":3,"method":"references","params":{"uri":"file:///a.tgo","offset":6,"includeDeclaration":true}}`,
			RequestTagReferences,
		},
		{
			"document symbols",
			`{"id":4,"method":"documentSymbols","params":{"uri":"file:///a.tgo"}}`,
			RequestTagDocumentSymbols,
		},
		{
			"workspace symbols",
			`{"id":5,"method":"workspaceSymbols","params":{"query":"Account"}}`,
			RequestTagWorkspaceSymbols,
		},
		{
			"cancel notification",
			`{"method":"cancel","params":{"id":5}}`,
			RequestTagCancel,
		},
		{
			"invalidate",
			`{"id":6,"method":"invalidate","params":{"uri":"file:///a.tgo"}}`,
			RequestTagInvalidate,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := RequestHover{Params: new(positionParams)}.Request()
			if err := json.Unmarshal([]byte(test.wire), &request); err != nil {
				t.Fatal(err)
			}
			if request.Tag() != test.tag {
				t.Fatalf("request tag = %v, want %v", request.Tag(), test.tag)
			}
			data, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			if !sameJSON(t, data, []byte(test.wire)) {
				t.Fatalf("request JSON = %s, want %s", data, test.wire)
			}
		})
	}
}

func TestRequestJSONRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		wire      string
		errorText string
	}{
		{"missing method", `{"id":1,"params":{}}`, "missing Request JSON tag"},
		{"unknown method", `{"id":1,"method":"rename","params":{}}`, "unknown Request JSON variant"},
		{"hover params", `{"id":1,"method":"hover","params":{"offset":"four"}}`, "cannot unmarshal"},
		{"definition params", `{"id":1,"method":"definition","params":false}`, "cannot unmarshal"},
		{"references params", `{"id":1,"method":"references","params":[]}`, "cannot unmarshal"},
		{"document params", `{"id":1,"method":"documentSymbols","params":{"uri":2}}`, "cannot unmarshal"},
		{"workspace params", `{"id":1,"method":"workspaceSymbols","params":{"query":true}}`, "cannot unmarshal"},
		{"cancel params", `{"id":1,"method":"cancel","params":{"id":"one"}}`, "cannot unmarshal"},
		{"invalidate params", `{"id":1,"method":"invalidate","params":{"uri":2}}`, "cannot unmarshal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := RequestHover{Params: new(positionParams)}.Request()
			err := json.Unmarshal([]byte(test.wire), &request)
			if err == nil || !strings.Contains(err.Error(), test.errorText) {
				t.Fatalf("error = %v, want text %q", err, test.errorText)
			}
		})
	}
}

func TestServeReportsProtocolErrors(t *testing.T) {
	engine, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	input := strings.NewReader(
		`{"id":1,"method":"rename","params":{}}` + "\n" +
			`{"id":2,"method":"hover"}` + "\n" +
			`{"id":3,"method":"definition"}` + "\n" +
			`{"id":4,"method":"references"}` + "\n" +
			`{"id":5,"method":"documentSymbols"}` + "\n" +
			`{"id":6,"method":"workspaceSymbols"}` + "\n" +
			`{"id":7,"method":"cancel"}` + "\n" +
			`{"id":8,"method":"invalidate"}` + "\n" +
			`{"id":9,"method":"cancel","params":{"id":"one"}}` + "\n",
	)
	var output bytes.Buffer
	if err := Serve(context.Background(), engine, input, &output); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	tests := []struct {
		id        int64
		errorText string
	}{
		{1, "unknown Request JSON variant"},
		{2, "missing params for hover"},
		{3, "missing params for definition"},
		{4, "missing params for references"},
		{5, "missing params for documentSymbols"},
		{6, "missing params for workspaceSymbols"},
		{7, "missing params for cancel"},
		{8, "missing params for invalidate"},
		{9, "cannot unmarshal"},
	}
	for _, test := range tests {
		var response Response
		if err := decoder.Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.ID != test.id || !strings.Contains(response.Error, test.errorText) {
			t.Fatalf("response = %#v, want ID %d and error %q", response, test.id, test.errorText)
		}
	}
}

func TestServeDoesNotReplyToInvalidCancelNotification(t *testing.T) {
	engine, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	input := strings.NewReader(
		`{"method":"cancel","params":{"id":"one"}}` + "\n" +
			`{"method":"cancel"}` + "\n",
	)
	var output bytes.Buffer
	if err := Serve(context.Background(), engine, input, &output); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("cancel notification output = %s, want empty output", &output)
	}
}

func sameJSON(t *testing.T, left, right []byte) bool {
	t.Helper()
	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		t.Fatal(err)
	}
	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(leftValue, rightValue)
}
