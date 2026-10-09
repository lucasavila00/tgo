package gaps

func consume(values ...any) {}

func lists() {
	consume(
		1,

		2,
	)
	consume(
		1,

	)
	_ = []int{
		1,

		2,

	}
	_ = struct {
		First int

		Second int

	}{
		First: 1,

		Second: 2,

	}
}

func block() {
	consume()

}

func commentGap() {
	consume()
	/* first line

	second line */
	consume()
}

func closingComments() {
	consume(
		1,
		// call close
	)
	_ = []int{
		1,
		// literal close
	}
	_ = struct {
		Value int
		// struct close
	}{
		Value: 1,
	}
	consume(
		// empty call
	)
	_ = []int{
		// empty literal
	}
}

type commentOnlyStruct struct {
	// empty struct
}
