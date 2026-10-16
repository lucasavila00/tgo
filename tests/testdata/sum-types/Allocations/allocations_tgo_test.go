package main

import "testing"

func BenchmarkSumTypes(b *testing.B) {
	b.Run("inline/TGo", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			result = makeSmallTGo(17)
		}
	})
	b.Run("inline/Go", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			result = makeSmallGo(17)
		}
	})
	b.Run("boxed/TGo", func(b *testing.B) {
		value := [256]byte{9}
		b.ReportAllocs()
		for b.Loop() {
			result = int(makeLargeTGo(value))
		}
	})
	b.Run("boxed/Go", func(b *testing.B) {
		value := [256]byte{9}
		b.ReportAllocs()
		for b.Loop() {
			result = int(makeLargeGo(value))
		}
	})
}
