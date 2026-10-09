package model

import (
	"encoding/json"
	"testing"
)

var benchmarkJSONData []byte

func BenchmarkEnumJSONMarshal(b *testing.B) {
	b.Run("zero", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b, JSONExternal{})
	})
	b.Run("external", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			JSONExternalCreated{ID: "a1"}.JSONExternal())
	})
	b.Run("internal", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			JSONInternalCreated{ID: "a1"}.JSONInternal())
	})
	b.Run("adjacent", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			JSONAdjacentCreated{ID: "a1"}.JSONAdjacent())
	})
	b.Run("untagged", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			JSONUntaggedText{Value: "text"}.JSONUntagged())
	})
	b.Run("escaped-external", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			JSONEscapedExternalValue{ID: "a1"}.JSONEscapedExternal())
	})
	b.Run("escaped-internal", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b, JSONEscapedValue{ID: "a1"}.JSONEscaped())
	})
	b.Run("escaped-adjacent", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			JSONEscapedAdjacentValue{ID: "a1"}.JSONEscapedAdjacent())
	})
}

func benchmarkEnumJSONMarshal[T any](b *testing.B, value T) {
	_, _ = json.Marshal(value)
	b.ResetTimer()
	var data []byte
	for b.Loop() {
		data, _ = json.Marshal(value)
	}
	benchmarkJSONData = data
}

func BenchmarkEnumJSONUnmarshal(b *testing.B) {
	tests := []struct {
		name string
		data []byte
		new  func() any
	}{
		{"zero", []byte(" \nnull\t"), func() any {
			return new(JSONExternal)
		}},
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
		{"escaped-external", []byte(`{"name\u0001\"end":{"id":"a1"}}`), func() any {
			return new(JSONEscapedExternal)
		}},
		{"escaped-internal", []byte(`{"kind\u0001":"name\u0001\"end","id":"a1"}`), func() any {
			return new(JSONEscaped)
		}},
		{"escaped-adjacent", []byte(`{"kind\u0001":"name\u0001\"end","data\u0002":{"id":"a1"}}`), func() any {
			return new(JSONEscapedAdjacent)
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
