package main

import "fmt"

func target() (int, error) {
	return 0, fmt.Errorf("failure")
}

func run() error {
	var error_tgo1 error
	_, error_tgo1 = target()
	if error_tgo1 != nil {
		return fmt.Errorf("target: %w", error_tgo1)
	}
	return nil
}

func main() {
	fmt.Println(run())
	fmt.Println(fmt_tgo3)
}
