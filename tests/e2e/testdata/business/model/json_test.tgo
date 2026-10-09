package model

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func TestEnumJSONForms(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		receiver any
		wire     string
	}{
		{"external", JSONExternalCreated{ID: "a1"}.JSONExternal(), new(JSONExternal), `{"created":{"account_id":"a1"}}`},
		{"internal", JSONInternalCreated{ID: "a1"}.JSONInternal(), new(JSONInternal), `{"type":"created","account_id":"a1"}`},
		{"adjacent", JSONAdjacentCreated{ID: "a1"}.JSONAdjacent(), new(JSONAdjacent), `{"type":"created","data":{"account_id":"a1"}}`},
		{"untagged number", JSONUntaggedNumber{Value: 42}.JSONUntagged(), new(JSONUntagged), `{"value":42}`},
		{"escaped", JSONEscapedValue{}.JSONEscaped(), new(JSONEscaped), `{"kind\u0001":"name\u0001\"end"}`},
		{"escaped external", JSONEscapedExternalValue{}.JSONEscapedExternal(), new(JSONEscapedExternal), `{"name\u0001\"end":{}}`},
		{"escaped adjacent", JSONEscapedAdjacentValue{}.JSONEscapedAdjacent(), new(JSONEscapedAdjacent), `{"kind\u0001":"name\u0001\"end","data\u0002":{}}`},
		{"string field", JSONStringFieldValue{Count: 42}.JSONStringField(), new(JSONStringField), `{"Value":{"count":"42"}}`},
		{"optional field", JSONExternalCreated{ID: "a1", Reason: "closed"}.JSONExternal(), new(JSONExternal), `{"created":{"account_id":"a1","reason":"closed"}}`},
		{"untagged", JSONUntaggedText{Value: "text"}.JSONUntagged(), new(JSONUntagged), `{"value":"text"}`},
		{"external empty", JSONExternalEmpty{}.JSONExternal(), new(JSONExternal), `{"Empty":{}}`},
		{"internal empty", JSONInternalEmpty{}.JSONInternal(), new(JSONInternal), `{"type":"Empty"}`},
		{"adjacent empty", JSONAdjacentEmpty{}.JSONAdjacent(), new(JSONAdjacent), `{"type":"Empty","data":{}}`},
		{"nested", JSONNestedNested{Value: JSONExternalCreated{ID: "a1"}.JSONExternal()}.JSONNested(), new(JSONNested), `{"Nested":{"value":{"created":{"account_id":"a1"}}}}`},
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
	external := JSONExternalCreated{ID: "old"}.JSONExternal()
	internal := JSONInternalCreated{ID: "old"}.JSONInternal()
	adjacent := JSONAdjacentCreated{ID: "old"}.JSONAdjacent()
	untagged := JSONUntaggedText{Value: "old"}.JSONUntagged()
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

func TestEnumJSONExternalDuplicateNames(t *testing.T) {
	var value JSONExternal
	if err := json.Unmarshal(
		[]byte(`{"created":{"account_id":7},"created":{"account_id":"last"}}`),
		&value,
	); err != nil {
		t.Fatal(err)
	}
	if value.Tag() != JSONExternalTagCreated || value.CreatedPayload().ID != "last" {
		t.Fatalf("last duplicate value was not selected: %#v", value)
	}

	value = (JSONExternalCreated{ID: "old"}).JSONExternal()
	err := json.Unmarshal(
		[]byte(`{"created":{"account_id":"first"},"created":{"account_id":7}}`),
		&value,
	)
	if err == nil {
		t.Fatal("invalid last duplicate value succeeded")
	}
	if value.Tag() != JSONExternalTagCreated || value.CreatedPayload().ID != "old" {
		t.Fatalf("failed duplicate decode changed the receiver: %#v", value)
	}

	for _, test := range []struct {
		input string
		want  string
	}{
		{`{"other":{},"other":{}}`, "unknown JSONExternal JSON variant"},
		{`{"created":{},"Empty":{}}`, "expected one JSONExternal JSON variant"},
	} {
		if err := json.Unmarshal([]byte(test.input), &value); err == nil ||
			!strings.Contains(err.Error(), test.want) {
			t.Fatalf("Unmarshal(%s) error = %v, want %q", test.input, err, test.want)
		}
	}
}

func TestEnumJSONDirectMethods(t *testing.T) {
	value := (JSONAdjacentCreated{ID: "a1"}).JSONAdjacent()
	data, err := value.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	const wire = `{"type":"created","data":{"account_id":"a1"}}`
	if string(data) != wire {
		t.Fatalf("MarshalJSON() = %s, want %s", data, wire)
	}
	var decoded JSONAdjacent
	if err := decoded.UnmarshalJSON([]byte(wire)); err != nil {
		t.Fatal(err)
	}
	if decoded.Tag() != JSONAdjacentTagCreated || decoded.CreatedPayload().ID != "a1" {
		t.Fatalf("UnmarshalJSON() = %#v", decoded)
	}
}

func TestEnumJSONAdjacentStreamMatchesDirectMethod(t *testing.T) {
	for _, input := range []string{
		`{"TYPE":"created","DATA":{"account_id":"folded"}}`,
		`{"type":"created","type":null,"data":{"account_id":"null"}}`,
		`{"type":"created","data":7,"data":{"account_id":"last"}}`,
		`{"type":7,"type":"created","data":{}}`,
		`{"type":"other","data":{}}`,
		`{"type":"created"}`,
	} {
		var streamed JSONAdjacent
		streamErr := json.Unmarshal([]byte(input), &streamed)
		var direct JSONAdjacent
		directErr := direct.UnmarshalJSON([]byte(input))
		if (streamErr == nil) != (directErr == nil) {
			t.Fatalf("Unmarshal(%s) stream error = %v, direct error = %v", input, streamErr, directErr)
		}
		if streamErr == nil && streamed != direct {
			t.Fatalf("Unmarshal(%s) stream = %#v, direct = %#v", input, streamed, direct)
		}
	}
}

func TestEnumJSONOrderAndPayloadRules(t *testing.T) {
	var value JSONUntagged
	if err := json.Unmarshal([]byte(`{"value":"text"}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.Tag() != JSONUntaggedTagText {
		t.Fatal("decode did not select the first matching variant")
	}
	if err := json.Unmarshal([]byte(`{}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.Tag() != JSONUntaggedTagNumber {
		t.Fatal("decode did not use declaration order")
	}
	var external JSONExternal
	if err := json.Unmarshal([]byte(`{"created":{"account_id":"a1","unknown":true}}`), &external); err != nil {
		t.Fatal(err)
	}
	if external.CreatedPayload().ID != "a1" {
		t.Fatal("wrong payload")
	}
	large := JSONExternalLarge{}.JSONExternal()
	data, err := json.Marshal(large)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &external); err != nil {
		t.Fatal(err)
	}
	if external.Tag() != large.Tag() || external.LargePayload() != large.LargePayload() {
		t.Fatal("boxed payload changed")
	}
	plain := JSONPlain{ID: "a1", Reason: ""}
	data, err = json.Marshal(plain)
	if err != nil || string(data) != `{"account_id":"a1"}` {
		t.Fatalf("plain struct: %s, %v", data, err)
	}
}

func TestEnumJSONCustomFields(t *testing.T) {
	value := JSONCustomValue{Value: "ok"}.JSONCustom()
	data, err := json.Marshal(value)
	if err != nil || string(data) != `{"Value":{"value":"custom:ok"}}` {
		t.Fatalf("custom field: %s, %v", data, err)
	}
	if err := json.Unmarshal([]byte(`{"Value":{"value":"decoded"}}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.ValuePayload().Value != "decoded" {
		t.Fatal("custom decoder was not used")
	}
	before := value
	if err := json.Unmarshal([]byte(`{"Value":{"value":"bad"}}`), &value); err == nil {
		t.Fatal("field error was lost")
	}
	if value != before {
		t.Fatal("receiver changed after field error")
	}
	if _, err := json.Marshal(JSONCustomValue{Value: "bad"}.JSONCustom()); err == nil {
		t.Fatal("field encode error was lost")
	}
}

func TestEnumJSONInvalidTags(t *testing.T) {
	zero := invalidJSONExternal(0)
	want := "JSONExternal: unknown tag 0 — tgolint proves every tag has a case, so this is unreachable"
	if zero.UnknownTag() != want {
		t.Fatalf("UnknownTag = %q", zero.UnknownTag())
	}
	for _, value := range []JSONExternal{zero, invalidJSONExternal(255)} {
		if _, err := json.Marshal(value); err == nil {
			t.Fatal("invalid tag was accepted")
		}
	}
}

func invalidJSONExternal(tag JSONExternalTag) JSONExternal {
	value := *new(JSONExternal)
	field := reflect.ValueOf(&value).Elem().FieldByName("tgoTag")
	writable := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	writable.SetUint(uint64(tag))
	return value
}

func TestEnumJSONCustomFieldsInTaggedForms(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		receiver  any
		wire, bad string
	}{
		{"external", JSONExternalCreated{Custom: "ok"}.JSONExternal(), new(JSONExternal), `{"created":{"custom":"ok"}}`, `{"created":{"custom":"bad"}}`},
		{"internal", JSONInternalCreated{Custom: "ok"}.JSONInternal(), new(JSONInternal), `{"type":"created","custom":"ok"}`, `{"type":"created","custom":"bad"}`},
		{"adjacent", JSONAdjacentCreated{Custom: "ok"}.JSONAdjacent(), new(JSONAdjacent), `{"type":"created","data":{"custom":"ok"}}`, `{"type":"created","data":{"custom":"bad"}}`},
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
	if external.Tag() != JSONExternalTagCreated || external.CreatedPayload() != (JSONExternalCreated{}) {
		t.Fatal("wrong null payload")
	}
	var adjacent JSONAdjacent
	if err := json.Unmarshal([]byte(`{"type":"created","data":null}`), &adjacent); err != nil {
		t.Fatal(err)
	}
	if adjacent.Tag() != JSONAdjacentTagCreated || adjacent.CreatedPayload() != (JSONAdjacentCreated{}) {
		t.Fatal("wrong null payload")
	}
	var untagged JSONUntagged
	if err := json.Unmarshal([]byte(`null`), &untagged); err != nil {
		t.Fatal(err)
	}
	if untagged.Tag() != JSONUntaggedTagNumber {
		t.Fatal("null did not select the first payload decode")
	}
}

func TestEnumJSONInternalPayloadMethods(t *testing.T) {
	direct := JSONInternalPayloadMethodValue{}.JSONInternalPayloadMethod()
	data, err := json.Marshal(direct)
	if err != nil || string(data) != `{"type":"value","custom":"payload"}` {
		t.Fatalf("direct method: %s, %v", data, err)
	}
	var decodedDirect JSONInternalPayloadMethod
	input := `{"type":"value","second":2,"first":1}`
	if err := json.Unmarshal([]byte(input), &decodedDirect); err != nil {
		t.Fatal(err)
	}
	if decodedDirect.ValuePayload().Seen != input {
		t.Fatalf("direct method input = %q", decodedDirect.ValuePayload().Seen)
	}

	promoted :=
		JSONInternalPromotedMethodValue{JSONObject: JSONObject{}}.JSONInternalPromotedMethod()

	data, err = json.Marshal(promoted)
	if err != nil || string(data) != `{"type":"value","custom":"promoted"}` {
		t.Fatalf("promoted method: %s, %v", data, err)
	}
	var decodedPromoted JSONInternalPromotedMethod
	input = `{"type":"value","last":2,"first":1}`
	if err := json.Unmarshal([]byte(input), &decodedPromoted); err != nil {
		t.Fatal(err)
	}
	if decodedPromoted.ValuePayload().Seen != input {
		t.Fatalf("promoted method input = %q", decodedPromoted.ValuePayload().Seen)
	}

	invalid :=
		JSONInternalPayloadMethodValue{Seen: "scalar"}.JSONInternalPayloadMethod()

	if _, err := json.Marshal(invalid); err == nil ||
		!strings.Contains(err.Error(), "expected JSONInternalPayloadMethod JSON payload object") {
		t.Fatalf("scalar payload error = %v", err)
	}
}
