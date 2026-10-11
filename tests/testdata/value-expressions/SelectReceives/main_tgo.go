package main

import "fmt"

func ready() chan int {
	channel := make(chan int, 1)
	channel <- 7
	return channel
}

func flowOnly(flag bool) int {
	channel := ready()
	{
		var tgo1 int
		if flag {
			select {
			case (<-channel):
				return 1
			}
			_ = 0 == 0
		} else {
			tgo1 = 2
		}
		return tgo1
	}

}

func bare(flag bool) int {
	channel := ready()
	var other chan int
	{
		var tgo2 int
		if flag {
			{
				var tgo3 chan int = channel
				var tgo4 chan int = other
				var tgo5 int
				if flag {
					tgo5 = 1
				} else {
					tgo5 = 2
				}
				select {
				case <-tgo3:
					return 7
				case tgo4 <- tgo5:
					return 99
				}
			}
			_ = 0 == 0
		} else {
			tgo2 = 2
		}
		return tgo2
	}

}

func assigned(flag bool) int {
	channel := ready()
	var other chan int
	value := 0
	{
		var tgo6 int
		if flag {
			{
				var tgo7 chan int = channel
				var tgo8 int
				var tgo9 chan int = other
				var tgo10 int
				if flag {
					tgo10 = 1
				} else {
					tgo10 = 2
				}
				select {
				case tgo8 = <-tgo7:
					value = tgo8
					return value
				case tgo9 <- tgo10:
					return 99
				}
			}
			_ = 0 == 0
		} else {
			tgo6 = 2
		}
		return tgo6
	}

}

func declared(flag bool) int {
	channel := ready()
	var other chan int
	{
		var tgo11 int
		if flag {
			{
				var tgo12 chan int = channel
				var tgo13 int
				var tgo14 bool
				var tgo15 chan int = other
				var tgo16 int
				if flag {
					tgo16 = 1
				} else {
					tgo16 = 2
				}
				select {
				case tgo13, tgo14 = <-tgo12:
					value, open := tgo13, tgo14
					if !open {
						panic("closed receive")
					}
					return value
				case tgo15 <- tgo16:
					return 99
				}
			}
			_ = 0 == 0
		} else {
			tgo11 = 2
		}
		return tgo11
	}

}

func main() {
	for _, flag := range []bool{true, false} {
		fmt.Println(flag, flowOnly(flag), bare(flag), assigned(flag), declared(flag))
	}
}
