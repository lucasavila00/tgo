package gaps

func cases(value any, ready <-chan int) {
	switch value {
	case nil:
		// empty case body
	case int:

	// next case description
	case string:
	case bool:
		// final empty body
	}

	select {
	case <-ready:
		// empty communication body
	default:

	// final clause description
	case <-ready:
		// final empty communication body
	}
}

func emptyClauses(ready <-chan int) {
	switch {
		// empty switch
	}
	select {
		// empty select
	}
}
