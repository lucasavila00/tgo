package gaps

type item struct {
	value int // last field

}

func opening(value int, ready <-chan int) {

	value++

	switch value {

	case 1:

		value++

		// next clause
	case 2:
		value++

		// body comment
	case 3:
	}

	select {

	case <-ready:

		value++
	default:
	}

	_ = []int{

		value,
		// last element

	}
}

func commentOnly() {
	_ = item{ /* value: 1 */ }
}
