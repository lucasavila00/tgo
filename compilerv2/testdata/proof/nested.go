package proof

func CaseNested(err error) error {
	call := func() error { effect("closure"); return nil }
	_ = call()
	effect("outside")
	return nil
}
