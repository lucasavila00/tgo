package main

import "testing"

var benchmarkValue int
var benchmarkError error

func BenchmarkPropagationReturn(b *testing.B) {
	for _, test := range []struct {
		name string
		fail bool
		wrap bool
	}{
		{"success", false, false},
		{"failure", true, false},
		{"wrapped failure", true, true},
	} {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkValue, benchmarkError = named(test.fail, test.wrap)
			}
		})
	}
}
