package main

import fmt_tgo4 "fmt"

import "fmt"

func target() (int, error) {
	return 0, fmt.Errorf("failure")
}

func run() error {
	{
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = target()
		if tgo2 != nil {
			return fmt_tgo4.Errorf("%s: %w", "target", tgo2)
		}
		_ = tgo1
	}
	return nil
}

func main() {
	fmt.Println(run())
	fmt.Println(fmt_tgo3)
}
