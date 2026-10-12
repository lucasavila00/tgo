package main

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
	var zero_tgo4 int
	defer fmt.Print("D")
	var tgo1 int
	if wrap {
		var value_tgo2 int
		var error_tgo3 error
		value_tgo2, error_tgo3 = next(fail)
		if error_tgo3 != nil {
			return zero_tgo4, fmt.Errorf("next: %w", error_tgo3)
		}
		tgo1 = value_tgo2
	} else {
		var value_tgo5 int
		var error_tgo6 error
		value_tgo5, error_tgo6 = next(fail)
		if error_tgo6 != nil {
			return zero_tgo4, error_tgo6
		}
		tgo1 = value_tgo5
	}
	value := tgo1

	return value, nil
}

func loop() int {
	total := 0
	for i := 0; i < 5; i++ {
		var tgo7 int
		if i == 1 {
			fmt.Print("C")
			continue
		} else if i == 3 {
			fmt.Print("B")
			break
		} else {
			tgo7 = i
		}
		value := tgo7

		total += value
	}
	return total
}

func jump(skip bool) int {
	value := 0
	{
		var tgo8 int
		if skip {
			goto done
		} else {
			tgo8 = 9
		}
		value = tgo8
	}

done:
	return value
}

func labelledValue() int { return 6 }

func tail() int {
	{
		var tgo9 int
		if true {
			goto value
		value:
			tgo9 = labelledValue()
		} else {
			tgo9 = 0
		}
		return tgo9
	}

}

func deferred() int {
	var tgo10 int
	if true {
		defer fmt.Print("F")
		tgo10 = 5
	} else {
		tgo10 = 0
	}
	return tgo10

}

func headers() int {
	total := 0
	{
		var tgo12 bool = 0 == 0
	condition:
		for i := 0; ; {
			if tgo12 {
				tgo12 = 0 != 0
			} else {

				i++
			}
			var tgo11 bool
			if i == 1 {
				break condition
			} else {
				tgo11 = i < 3
			}
			if !tgo11 {
				break
			}
			{
				total += i
			}
		}
	}
	checks := 0
	{
		var tgo14 bool = 0 == 0
	retry:
		for i := 0; ; {
			if tgo14 {
				tgo14 = 0 != 0
			} else {

				i++
			}
			var tgo13 bool
			if checks++; checks == 1 {
				continue retry
			} else {
				tgo13 = checks < 3
			}
			if !tgo13 {
				break
			}
			{
				total += i
			}
		}
	}
	{
		var tgo16 bool = 0 == 0
	post:
		for i := 0; ; {
			if tgo16 {
				tgo16 = 0 != 0
			} else {
				var tgo15 int
				if i == 1 {
					break post
				} else {
					tgo15 = 1
				}
				i += tgo15
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
			var tgo17 int
			if i == 1 {
				continue
			} else if i == 3 {
				break
			} else {
				tgo17 = i
			}
			value := tgo17
			if value >= 0 {
				total += value
			}
		}
	}
	for i := 0; i < 5; i++ {
		var tgo18 int
		{
			var tgo19 int
			if i == 1 {
				continue
			} else if i == 3 {
				break
			} else {
				tgo19 = i
			}
			value := tgo19
			if value >= 0 {
				tgo18 = value
			} else {
				tgo18 = 0
			}
		}
		total += tgo18

	}
	return total
}

func labelledLoop() int {
	turns := 0
again:
	{
		var tgo22 int
		if turns < 2 {
			turns++
			tgo22 = 0
		} else {
			tgo22 = 0
		}
		var tgo23 bool = 0 == 0
	tgo20:
		for i := tgo22; ; {
			if tgo23 {
				tgo23 = 0 != 0
			} else {

				i++
			}
			if !(i < 1) {
				break
			}
			{
				{
					var tgo21 int
					if turns < 2 {
						goto again
					} else {
						break tgo20
					}
					_ = int(tgo21)
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
