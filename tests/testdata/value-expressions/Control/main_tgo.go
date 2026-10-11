package main

import fmt_tgo5 "fmt"

import (
	"errors"
	"fmt"
)

var failure = errors.New("failure")

func next(fail bool) (int, error) {
	fmt.Print("N")
	if fail {
		return 0, failure
	}
	return 7, nil
}

func checked(wrap, fail bool) (int, error) {
	var tgo4 int
	defer fmt.Print("D")
	var tgo1 int
	if wrap {
		var tgo2 int
		var tgo3 error
		tgo2, tgo3 = next(fail)
		if tgo3 != nil {
			return tgo4, fmt_tgo5.Errorf("next: %w", tgo3)
		}
		tgo1 = tgo2
	} else {
		var tgo6 int
		var tgo7 error
		tgo6, tgo7 = next(fail)
		if tgo7 != nil {
			return tgo4, tgo7
		}
		tgo1 = tgo6
	}
	value := tgo1

	return value, nil
}

func loop() int {
	total := 0
	for i := 0; i < 5; i++ {
		var tgo8 int
		if i == 1 {
			fmt.Print("C")
			continue
		} else if i == 3 {
			fmt.Print("B")
			break
		} else {
			tgo8 = i
		}
		value := tgo8

		total += value
	}
	return total
}

func jump(skip bool) int {
	value := 0
	{
		var tgo9 int
		if skip {
			goto done
		} else {
			tgo9 = 9
		}
		value = tgo9
	}

done:
	return value
}

func labelledValue() int { return 6 }

func tail() int {
	{
		var tgo10 int
		if true {
			goto value
		value:
			tgo10 = labelledValue()
		} else {
			tgo10 = 0
		}
		return tgo10
	}

}

func deferred() int {
	{
		var tgo11 int
		if true {
			defer fmt.Print("F")
			tgo11 = 5
		} else {
			tgo11 = 0
		}
		return tgo11
	}

}

func headers() int {
	total := 0
	{
		var tgo13 bool = 0 == 0
	condition:
		for i := 0; ; {
			if tgo13 {
				tgo13 = 0 != 0
			} else {

				i++
			}
			var tgo12 bool
			if i == 1 {
				break condition
			} else {
				tgo12 = i < 3
			}
			if !tgo12 {
				break
			}
			{
				total += i
			}
		}
	}
	checks := 0
	{
		var tgo15 bool = 0 == 0
	retry:
		for i := 0; ; {
			if tgo15 {
				tgo15 = 0 != 0
			} else {

				i++
			}
			var tgo14 bool
			if checks++; checks == 1 {
				continue retry
			} else {
				tgo14 = checks < 3
			}
			if !tgo14 {
				break
			}
			{
				total += i
			}
		}
	}
	{
		var tgo17 bool = 0 == 0
	post:
		for i := 0; ; {
			if tgo17 {
				tgo17 = 0 != 0
			} else {
				var tgo16 int
				if i == 1 {
					break post
				} else {
					tgo16 = 1
				}
				i += tgo16
			}
			if !(i < 3) {
				break
			}

			{
				total += i
			}
		}
	}
	for i := 0; i < 5; i++ {
		{
			var tgo18 int
			if i == 1 {
				continue
			} else if i == 3 {
				break
			} else {
				tgo18 = i
			}
			value := tgo18
			if value >= 0 {
				total += value
			}
		}
	}
	for i := 0; i < 5; i++ {
		{
			var tgo19 int
			{
				var tgo20 int
				if i == 1 {
					continue
				} else if i == 3 {
					break
				} else {
					tgo20 = i
				}
				value := tgo20
				if value >= 0 {
					tgo19 = value
				} else {
					tgo19 = 0
				}
			}
			total += tgo19
		}

	}
	return total
}

func labelledLoop() int {
	turns := 0
again:
	{
		{
			var tgo23 int
			if turns < 2 {
				turns++
				tgo23 = 0
			} else {
				tgo23 = 0
			}
			var tgo24 bool = 0 == 0
		tgo21:
			for i := tgo23; ; {
				if tgo24 {
					tgo24 = 0 != 0
				} else {

					i++
				}
				if !(i < 1) {
					break
				}
				{
					{
						var tgo22 int
						if turns < 2 {
							goto again
						} else {
							break tgo21
						}
						_ = int(tgo22)
					}
				}
			}
		}
	}
	return turns
}

func main() {
	if labelledLoop() != 2 {
		panic("loop label")
	}
	if headers() != 6 {
		panic("header jump target")
	}
	if tail() != 6 {
		panic("labelled value")
	}
	value, err := checked(false, false)
	if value != 7 || err != nil {
		panic("success")
	}
	fmt.Print(";")
	value, err = checked(false, true)
	if value != 0 || err != failure {
		panic("direct error identity")
	}
	fmt.Print(";")
	value, err = checked(true, true)
	if value != 0 || !errors.Is(err, failure) || err.Error() != "next: failure" {
		panic("wrapped error")
	}
	fmt.Print(";")
	fmt.Print("loop=", loop(), ";jump=", jump(true), ",", jump(false), ";defer=", deferred(), "\n")
}
