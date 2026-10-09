// Package variantflow models the possible variants of one enum value.
package variantflow

import "math/bits"

// Type is a union of enum variant tags from 1 through 255.
type Type [4]uint64

// Never returns the empty union.
func Never() Type { return Type{} }

// Variant returns the type that contains one variant tag.
func Variant(tag int) Type {
	var result Type
	index := tag - 1
	result[index/64] = uint64(1) << (index % 64)
	return result
}

// All returns the union of all declared variant tags.
func All(count int) Type {
	result := Never()
	fullWords := count / 64
	for index := 0; index < fullWords; index++ {
		result[index] = ^uint64(0)
	}
	if remaining := count % 64; remaining != 0 {
		result[fullWords] = uint64(1)<<remaining - 1
	}
	return result
}

// Union returns the variants that occur in either type.
func Union(left, right Type) Type {
	var result Type
	for index := range result {
		result[index] = left[index] | right[index]
	}
	return result
}

// Intersect returns the variants that occur in both types.
func Intersect(left, right Type) Type {
	var result Type
	for index := range result {
		result[index] = left[index] & right[index]
	}
	return result
}

// Without removes every right variant from left.
func Without(left, right Type) Type {
	var result Type
	for index := range result {
		result[index] = left[index] &^ right[index]
	}
	return result
}

// IsNever reports whether no variant is possible.
func IsNever(value Type) bool { return value == Never() }

// Singleton returns the only possible tag, if there is one.
func Singleton(value Type) (int, bool) {
	tag := 0
	for index, word := range value {
		if word == 0 {
			continue
		}
		if bits.OnesCount64(word) != 1 || tag != 0 {
			return 0, false
		}
		tag = index*64 + bits.TrailingZeros64(word) + 1
	}
	return tag, tag != 0
}

// Tags returns the possible tags in numeric order.
func Tags(value Type) []int {
	count := 0
	for _, word := range value {
		count += bits.OnesCount64(word)
	}
	result := make([]int, 0, count)
	for index, word := range value {
		for word != 0 {
			bit := bits.TrailingZeros64(word)
			result = append(result, index*64+bit+1)
			word &= word - 1
		}
	}
	return result
}
