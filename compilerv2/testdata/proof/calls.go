package proof

func CaseCalls(err error) error {
	consume(effect("left"), effect("right"))
	consume(pair())
	return nil
}
