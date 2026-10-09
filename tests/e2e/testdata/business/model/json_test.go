package model

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestEnumJSONForms(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		receiver any
		wire     string
	}{
		{"external", NewJSONExternalCreated(JSONExternalCreated{ID: "a1"}), new(JSONExternal), `{"created":{"account_id":"a1"}}`},
		{"internal", NewJSONInternalCreated(JSONInternalCreated{ID: "a1"}), new(JSONInternal), `{"type":"created","account_id":"a1"}`},
		{"adjacent", NewJSONAdjacentCreated(JSONAdjacentCreated{ID: "a1"}), new(JSONAdjacent), `{"type":"created","data":{"account_id":"a1"}}`},
		{"untagged number", NewJSONUntaggedNumber(JSONUntaggedNumber{Value: 42}), new(JSONUntagged), `{"value":42}`},
		{"escaped", NewJSONEscapedValue(JSONEscapedValue{}), new(JSONEscaped), `{"kind\u0001":"name\u0001\"end"}`},
		{"escaped external", NewJSONEscapedExternalValue(JSONEscapedExternalValue{}), new(JSONEscapedExternal), `{"name\u0001\"end":{}}`},
		{"escaped adjacent", NewJSONEscapedAdjacentValue(JSONEscapedAdjacentValue{}), new(JSONEscapedAdjacent), `{"kind\u0001":"name\u0001\"end","data\u0002":{}}`},
		{"string field", NewJSONStringFieldValue(JSONStringFieldValue{Count: 42}), new(JSONStringField), `{"Value":{"count":"42"}}`},
		{"optional field", NewJSONExternalCreated(JSONExternalCreated{ID: "a1", Reason: "closed"}), new(JSONExternal), `{"created":{"account_id":"a1","reason":"closed"}}`},
		{"untagged", NewJSONUntaggedText(JSONUntaggedText{Value: "text"}), new(JSONUntagged), `{"value":"text"}`},
		{"external empty", NewJSONExternalEmpty(JSONExternalEmpty{}), new(JSONExternal), `{"Empty":{}}`},
		{"internal empty", NewJSONInternalEmpty(JSONInternalEmpty{}), new(JSONInternal), `{"type":"Empty"}`},
		{"adjacent empty", NewJSONAdjacentEmpty(JSONAdjacentEmpty{}), new(JSONAdjacent), `{"type":"Empty","data":{}}`},
		{"nested", NewJSONNestedNested(JSONNestedNested{Value: NewJSONExternalCreated(JSONExternalCreated{ID: "a1"})}), new(JSONNested), `{"Nested":{"value":{"created":{"account_id":"a1"}}}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.value)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.wire), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("wire = %s, want %s", data, test.wire)
			}
			if err := json.Unmarshal([]byte(test.wire), test.receiver); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(reflect.ValueOf(test.receiver).Elem().Interface(), test.value) {
				t.Fatalf("decoded = %#v, want %#v", test.receiver, test.value)
			}
		})
	}
}

func TestEnumJSONDecodeFailureKeepsReceiver(t *testing.T) {
	external := NewJSONExternalCreated(JSONExternalCreated{ID: "old"})
	internal := NewJSONInternalCreated(JSONInternalCreated{ID: "old"})
	adjacent := NewJSONAdjacentCreated(JSONAdjacentCreated{ID: "old"})
	untagged := NewJSONUntaggedText(JSONUntaggedText{Value: "old"})
	tests := []struct {
		name     string
		receiver any
		inputs   []string
	}{
		{"external", &external, []string{`null`, `[]`, `0`, `{}`, `{"unknown":{}}`, `{"Empty":{},"created":{}}`, `{"created":[]}`, `{"created":{"account_id":1}}`, `{`}},
		{"internal", &internal, []string{`null`, `[]`, `{}`, `{"type":"unknown"}`, `{"type":1}`, `{"type":null}`, `{"type":"created","account_id":1}`}},
		{"adjacent", &adjacent, []string{`null`, `{}`, `{"type":"unknown","data":{}}`, `{"type":"created"}`, `{"type":1,"data":{}}`, `{"type":"created","data":[]}`, `{"type":"created","data":{"account_id":1}}`}},
		{"untagged", &untagged, []string{`[]`, `0`, `{"value":[]}`}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := reflect.ValueOf(test.receiver).Elem().Interface()
			for _, input := range test.inputs {
				if err := json.Unmarshal([]byte(input), test.receiver); err == nil {
					t.Errorf("accepted %s", input)
				}
				if !reflect.DeepEqual(reflect.ValueOf(test.receiver).Elem().Interface(), before) {
					t.Fatalf("receiver changed after %s", input)
				}
			}
		})
	}
}

func TestEnumJSONOrderAndPayloadRules(t *testing.T) {
	var value JSONUntagged
	if err := json.Unmarshal([]byte(`{"value":"text"}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.TgoTag() != 2 {
		t.Fatal("decode did not select the first matching variant")
	}
	if err := json.Unmarshal([]byte(`{}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.TgoTag() != 1 {
		t.Fatal("decode did not use declaration order")
	}
	var external JSONExternal
	if err := json.Unmarshal([]byte(`{"created":{"account_id":"a1","unknown":true}}`), &external); err != nil {
		t.Fatal(err)
	}
	if external.TgoCreated().ID != "a1" {
		t.Fatal("wrong payload")
	}
	large := NewJSONExternalLarge(JSONExternalLarge{})
	data, err := json.Marshal(large)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &external); err != nil {
		t.Fatal(err)
	}
	if external.TgoTag() != large.TgoTag() || external.TgoLarge() != large.TgoLarge() {
		t.Fatal("boxed payload changed")
	}
	plain := JSONPlain{ID: "a1", Reason: ""}
	data, err = json.Marshal(plain)
	if err != nil || string(data) != `{"account_id":"a1"}` {
		t.Fatalf("plain struct: %s, %v", data, err)
	}
}

func TestEnumJSONCustomFields(t *testing.T) {
	value := NewJSONCustomValue(JSONCustomValue{Value: "ok"})
	data, err := json.Marshal(value)
	if err != nil || string(data) != `{"Value":{"value":"custom:ok"}}` {
		t.Fatalf("custom field: %s, %v", data, err)
	}
	if err := json.Unmarshal([]byte(`{"Value":{"value":"decoded"}}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.TgoValue().Value != "decoded" {
		t.Fatal("custom decoder was not used")
	}
	before := value
	if err := json.Unmarshal([]byte(`{"Value":{"value":"bad"}}`), &value); err == nil {
		t.Fatal("field error was lost")
	}
	if value != before {
		t.Fatal("receiver changed after field error")
	}
	if _, err := json.Marshal(NewJSONCustomValue(JSONCustomValue{Value: "bad"})); err == nil {
		t.Fatal("field encode error was lost")
	}
}

func TestEnumJSONInvalidTags(t *testing.T) {
	for _, value := range []JSONExternal{{}, {tgoTag: 255}} {
		if _, err := json.Marshal(value); err == nil {
			t.Fatal("invalid tag was accepted")
		}
	}
}

func TestEnumJSONCustomFieldsInTaggedForms(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		receiver  any
		wire, bad string
	}{
		{"external", NewJSONExternalCreated(JSONExternalCreated{Custom: "ok"}), new(JSONExternal), `{"created":{"custom":"ok"}}`, `{"created":{"custom":"bad"}}`},
		{"internal", NewJSONInternalCreated(JSONInternalCreated{Custom: "ok"}), new(JSONInternal), `{"type":"created","custom":"ok"}`, `{"type":"created","custom":"bad"}`},
		{"adjacent", NewJSONAdjacentCreated(JSONAdjacentCreated{Custom: "ok"}), new(JSONAdjacent), `{"type":"created","data":{"custom":"ok"}}`, `{"type":"created","data":{"custom":"bad"}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), `"custom":"custom:ok"`) {
				t.Fatalf("custom encoder: %s", data)
			}
			if err := json.Unmarshal([]byte(test.wire), test.receiver); err != nil {
				t.Fatal(err)
			}
			before := reflect.ValueOf(test.receiver).Elem().Interface()
			if !reflect.DeepEqual(before, test.value) {
				t.Fatal("custom decoder was not used")
			}
			if err := json.Unmarshal([]byte(test.bad), test.receiver); err == nil {
				t.Fatal("custom field error was lost")
			}
			if !reflect.DeepEqual(reflect.ValueOf(test.receiver).Elem().Interface(), before) {
				t.Fatal("receiver changed after field error")
			}
		})
	}
}

func TestEnumJSONNullPayloads(t *testing.T) {
	var external JSONExternal
	if err := json.Unmarshal([]byte(`{"created":null}`), &external); err != nil {
		t.Fatal(err)
	}
	if external.TgoTag() != 1 || external.TgoCreated() != (JSONExternalCreated{}) {
		t.Fatal("wrong null payload")
	}
	var adjacent JSONAdjacent
	if err := json.Unmarshal([]byte(`{"type":"created","data":null}`), &adjacent); err != nil {
		t.Fatal(err)
	}
	if adjacent.TgoTag() != 1 || adjacent.TgoCreated() != (JSONAdjacentCreated{}) {
		t.Fatal("wrong null payload")
	}
	var untagged JSONUntagged
	if err := json.Unmarshal([]byte(`null`), &untagged); err != nil {
		t.Fatal(err)
	}
	if untagged.TgoTag() != 1 {
		t.Fatal("null did not select the first payload decode")
	}
}

func TestEnumJSONInternalPayloadMethods(t *testing.T) {
	direct := NewJSONInternalPayloadMethodValue(JSONInternalPayloadMethodValue{})
	data, err := json.Marshal(direct)
	if err != nil || string(data) != `{"type":"value","custom":"payload"}` {
		t.Fatalf("direct method: %s, %v", data, err)
	}
	var decodedDirect JSONInternalPayloadMethod
	input := `{"type":"value","second":2,"first":1}`
	if err := json.Unmarshal([]byte(input), &decodedDirect); err != nil {
		t.Fatal(err)
	}
	if decodedDirect.TgoValue().Seen != input {
		t.Fatalf("direct method input = %q", decodedDirect.TgoValue().Seen)
	}

	promoted := NewJSONInternalPromotedMethodValue(
		JSONInternalPromotedMethodValue{JSONObject: JSONObject{}},
	)
	data, err = json.Marshal(promoted)
	if err != nil || string(data) != `{"type":"value","custom":"promoted"}` {
		t.Fatalf("promoted method: %s, %v", data, err)
	}
	var decodedPromoted JSONInternalPromotedMethod
	input = `{"type":"value","last":2,"first":1}`
	if err := json.Unmarshal([]byte(input), &decodedPromoted); err != nil {
		t.Fatal(err)
	}
	if decodedPromoted.TgoValue().Seen != input {
		t.Fatalf("promoted method input = %q", decodedPromoted.TgoValue().Seen)
	}

	invalid := NewJSONInternalPayloadMethodValue(
		JSONInternalPayloadMethodValue{Seen: "scalar"},
	)
	if _, err := json.Marshal(invalid); err == nil ||
		!strings.Contains(err.Error(), "expected JSONInternalPayloadMethod JSON payload object") {
		t.Fatalf("scalar payload error = %v", err)
	}
}
