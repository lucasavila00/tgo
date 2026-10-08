package model

import (
	"encoding/json"
	"testing"
)

var benchmarkJSONData []byte

func BenchmarkEnumJSONMarshal(b *testing.B) {
	values := []struct {
		name  string
		value any
	}{
		{"external", NewJSONExternalCreated(JSONExternalCreated{ID: "a1"})},
		{"internal", NewJSONInternalCreated(JSONInternalCreated{ID: "a1"})},
		{"adjacent", NewJSONAdjacentCreated(JSONAdjacentCreated{ID: "a1"})},
		{"untagged", NewJSONUntaggedText(JSONUntaggedText{Value: "text"})},
	}
	for _, test := range values {
		b.Run(test.name, func(b *testing.B) {
			_, _ = json.Marshal(test.value)
			b.ResetTimer()
			var data []byte
			for b.Loop() {
				data, _ = json.Marshal(test.value)
			}
			benchmarkJSONData = data
		})
	}
}

func BenchmarkEnumJSONUnmarshal(b *testing.B) {
	tests := []struct {
		name string
		data []byte
		new  func() any
	}{
		{"external", []byte(`{"created":{"account_id":"a1"}}`), func() any {
			return new(JSONExternal)
		}},
		{"internal", []byte(`{"type":"created","account_id":"a1"}`), func() any {
			return new(JSONInternal)
		}},
		{"adjacent", []byte(`{"type":"created","data":{"account_id":"a1"}}`), func() any {
			return new(JSONAdjacent)
		}},
		{"untagged", []byte(`{"value":"text"}`), func() any {
			return new(JSONUntagged)
		}},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			value := test.new()
			_ = json.Unmarshal(test.data, value)
			b.ResetTimer()
			for b.Loop() {
				_ = json.Unmarshal(test.data, value)
			}
		})
	}
}
