package proof

func CaseScheduling(err error) error {
	defer consume(effect("defer argument"))
	done := make(chan struct{})
	go func(value int) { close(done) }(effect("go argument"))
	<-done
	effect("after wait")
	return nil
}

func CaseRecover(err error) error {
	defer func() {
		if recover() != nil {
			effect("recovered")
		}
	}()
	panic(effect("panic argument"))
}
