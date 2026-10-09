package model

import (
	"encoding/json"
	"testing"
)

var benchmarkJSONData []byte

func BenchmarkEnumJSONMarshal(b *testing.B) {
	b.Run("external", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			NewJSONExternalCreated(JSONExternalCreated{ID: "a1"}))
	})
	b.Run("internal", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			NewJSONInternalCreated(JSONInternalCreated{ID: "a1"}))
	})
	b.Run("adjacent", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			NewJSONAdjacentCreated(JSONAdjacentCreated{ID: "a1"}))
	})
	b.Run("untagged", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			NewJSONUntaggedText(JSONUntaggedText{Value: "text"}))
	})
	b.Run("escaped-external", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			NewJSONEscapedExternalValue(JSONEscapedExternalValue{ID: "a1"}))
	})
	b.Run("escaped-internal", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b, NewJSONEscapedValue(JSONEscapedValue{ID: "a1"}))
	})
	b.Run("escaped-adjacent", func(b *testing.B) {
		benchmarkEnumJSONMarshal(b,
			NewJSONEscapedAdjacentValue(JSONEscapedAdjacentValue{ID: "a1"}))
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

func BenchmarkEnumJSONMarshalMethod(b *testing.B) {
	b.Run("external", func(b *testing.B) {
		benchmarkEnumJSONMarshalMethod(b,
			NewJSONExternalCreated(JSONExternalCreated{ID: "a1"}))
	})
	b.Run("internal", func(b *testing.B) {
		benchmarkEnumJSONMarshalMethod(b,
			NewJSONInternalCreated(JSONInternalCreated{ID: "a1"}))
	})
	b.Run("adjacent", func(b *testing.B) {
		benchmarkEnumJSONMarshalMethod(b,
			NewJSONAdjacentCreated(JSONAdjacentCreated{ID: "a1"}))
	})
	b.Run("untagged", func(b *testing.B) {
		benchmarkEnumJSONMarshalMethod(b,
			NewJSONUntaggedText(JSONUntaggedText{Value: "text"}))
	})
	b.Run("escaped-external", func(b *testing.B) {
		benchmarkEnumJSONMarshalMethod(b,
			NewJSONEscapedExternalValue(JSONEscapedExternalValue{ID: "a1"}))
	})
	b.Run("escaped-internal", func(b *testing.B) {
		benchmarkEnumJSONMarshalMethod(b,
			NewJSONEscapedValue(JSONEscapedValue{ID: "a1"}))
	})
	b.Run("escaped-adjacent", func(b *testing.B) {
		benchmarkEnumJSONMarshalMethod(b,
			NewJSONEscapedAdjacentValue(JSONEscapedAdjacentValue{ID: "a1"}))
	})
}

func benchmarkEnumJSONMarshalMethod[
	T interface{ MarshalJSON() ([]byte, error) },
](b *testing.B, value T) {
	_, _ = value.MarshalJSON()
	b.ResetTimer()
	var data []byte
	for b.Loop() {
		data, _ = value.MarshalJSON()
	}
	benchmarkJSONData = data
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

func BenchmarkEnumJSONUnmarshalMethod(b *testing.B) {
	tests := []struct {
		name string
		data []byte
		new  func() interface{ UnmarshalJSON([]byte) error }
	}{
		{"external", []byte(`{"created":{"account_id":"a1"}}`), func() interface{ UnmarshalJSON([]byte) error } {
			return new(JSONExternal)
		}},
		{"internal", []byte(`{"type":"created","account_id":"a1"}`), func() interface{ UnmarshalJSON([]byte) error } {
			return new(JSONInternal)
		}},
		{"adjacent", []byte(`{"type":"created","data":{"account_id":"a1"}}`), func() interface{ UnmarshalJSON([]byte) error } {
			return new(JSONAdjacent)
		}},
		{"untagged", []byte(`{"value":"text"}`), func() interface{ UnmarshalJSON([]byte) error } {
			return new(JSONUntagged)
		}},
		{"escaped-external", []byte(`{"name\u0001\"end":{"id":"a1"}}`), func() interface{ UnmarshalJSON([]byte) error } {
			return new(JSONEscapedExternal)
		}},
		{"escaped-internal", []byte(`{"kind\u0001":"name\u0001\"end","id":"a1"}`), func() interface{ UnmarshalJSON([]byte) error } {
			return new(JSONEscaped)
		}},
		{"escaped-adjacent", []byte(`{"kind\u0001":"name\u0001\"end","data\u0002":{"id":"a1"}}`), func() interface{ UnmarshalJSON([]byte) error } {
			return new(JSONEscapedAdjacent)
		}},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			value := test.new()
			_ = value.UnmarshalJSON(test.data)
			b.ResetTimer()
			for b.Loop() {
				_ = value.UnmarshalJSON(test.data)
			}
		})
	}
}
