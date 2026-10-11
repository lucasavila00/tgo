package proof

func CaseRanges(err error) error {
	for i := range (*[2]int)(nil) {
		consume(i)
	}
	var a [3]int
	for _, a[effect("range target")] = range []int{1, 2} {
		effect("range body")
	}
	return nil
}
