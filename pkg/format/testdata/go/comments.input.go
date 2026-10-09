package parity

func use(values ...int) {}

func comments() {
	use(0 /* comma, inside */, 1)
	use(0, /* after comma */ 1)
	use(0 /* before close */)

	_ = []int{0 /* comma, inside */, 1}
	_ = []int{0, /* after comma */ 1}
	_ = []int{0 /* before close */}
}
