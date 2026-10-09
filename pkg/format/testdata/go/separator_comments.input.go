package separators

func consume(values ...int) {}

func separators(first, second int, values []int) {
	consume(first /* before */, second)
	consume(first, /* after */ second)
	consume(first /* left */, /* right */ second)
	consume(first /* before one */ /* before two */, second)
	consume(first, /* after one */ /* after two */ second)

	consume(
		first /* before */,
		second,
	)
	consume(
		first, /* after */
		second,
	)
	consume(
		first /* left */, /* right one */ /* right two */
		second,
	)
	consume(
		first, /* block */ // line
		second,
	)

	consume(first, /* call close */)
	_ = values[first /* index close */]
	_ = []int{first, /* literal close */}
	_ = []int{
		first /* left */, /* right */
		second,
	}
}
