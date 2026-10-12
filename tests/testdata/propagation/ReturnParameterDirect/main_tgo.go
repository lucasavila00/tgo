package main

type Failure interface{ Error() string }

func next() (int, Failure) { return 7, nil }

func run(error int) error {
	var error_tgo1 Failure
	_, error_tgo1 = next()
	if error_tgo1 != nil {
		return error_tgo1
	}
	return nil
}

func main() {
	if run(0) != nil {
		panic("unexpected error")
	}
}
