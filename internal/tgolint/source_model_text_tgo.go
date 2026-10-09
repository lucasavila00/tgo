package tgolint

import "tgo/pkg/format"

func sameTypeText(left string, right string) bool {
	return sameFormattedText(
		left,
		right,
		func(text string) string {
			return "package p\ntype value " + text + "\n"
		},
	)
}

func sameTagText(left string, right string) bool {
	return sameFormattedText(
		left,
		right,
		func(text string) string {
			return "package p\ntype value struct { Field int " + text + " }\n"
		},
	)
}

func sameFormattedText(
	left string,
	right string,
	wrap func(string) string,
) bool {
	if left == right {
		return true
	}
	if left == "" || right == "" {
		return left == right
	}
	leftText, leftErr := format.Source("left.tgo", []byte(wrap(left)))
	rightText, rightErr := format.Source("right.tgo", []byte(wrap(right)))
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftText) == string(rightText)
}
