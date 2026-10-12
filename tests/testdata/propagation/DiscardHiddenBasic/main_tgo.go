package main

func get() (int, error) { return 1, nil }

func run(int string) error {
	var error_tgo1 error
	_, error_tgo1 = get()
	if error_tgo1 != nil {
		return error_tgo1
	}
	return nil
}

func main() {
	if run("") != nil {
		panic("failed")
	}
}
