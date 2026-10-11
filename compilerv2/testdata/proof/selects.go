package proof

func CaseSelects(err error) error {
	ready := make(chan int, 1)
	ready <- effect("send")
	var a [3]int
	select {
	case a[effect("receive target")] = <-ready:
		effect("received")
	default:
		effect("unexpected default")
	}
	return nil
}
