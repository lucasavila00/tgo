package main

type Failure interface{ Error() string }

func next() (int, Failure) { return 7, nil }

func run() (error error) {
	{
		var tgo1 int
		var tgo2 Failure
		tgo1, tgo2 = next()
		if tgo2 != nil {
			return tgo2
		}
		_ = tgo1
	}
	return nil
}

func main() {
	if run() != nil {
		panic("unexpected error")
	}
}
