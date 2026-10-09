package app

func resultFromLoop(run bool) (result int) {
	for result = 1; run; run = false {
	}
	return
}

func resultFromSwitch(value int) (result int) {
	switch result = 1; value {
	case 1:
		result = 2
	}
	return
}

func resultFromSelect(values <-chan int) (result int) {
	result = 1
	select {
	case result = <-values:
	default:
	}
	return
}

func resultFromLiteral() (result int) {
	result = 1
	read := func() int {
		return result
	}
	return read()
}

func literalResult() int {
	calculate := func() (result int) {
		result = 3
		return
	}
	return calculate()
}

func writeInLiteral() (result int) {
	write := func() {
		result = 1
	}
	_ = write
	result = 2
	return
}

func returnBeforePost(run bool) (result int) {
	for ; run; _ = result {
		return 1
	}
	result = 2
	return
}

func resultBeforeGoto() (result int) {
	result = 1
	goto done
done:
	return
}

func nestedContinue(run bool) (result int) {
	for ; run; _ = result {
		for false {
			continue
		}
		return 1
	}
	result = 2
	return
}

func outerContinue(run bool) (result int) {
outer:
	for ; run; run = false {
		for ; run; _ = result {
			continue outer
		}
		return 1
	}
	result = 2
	return
}

func outerBreak(run bool) (result int) {
outer:
	for ; run; _ = result {
		for {
			break outer
		}
	}
	result = 2
	return
}

func unreachableContinue(run bool) (result int) {
	for ; run; _ = result {
		return 1
		continue
	}
	result = 2
	return
}

func switchFallthrough(run bool) (result int) {
	for ; run; _ = result {
		switch 1 {
		case 1:
			fallthrough
		default:
			return 1
		}
	}
	result = 2
	return
}
