package proof

import "sync"

func CaseScheduling(err error) error {
	defer consume(effect("defer argument"))
	done := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(1)
	go func(value int) { defer workers.Done(); close(done) }(effect("go argument"))
	<-done
	workers.Wait()
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
