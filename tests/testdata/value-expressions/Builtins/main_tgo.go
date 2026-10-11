package main

import "fmt"

func choose(ok bool) int8 {
	{
		var tgo1 int8
		if ok {
			return 7
		} else {
			return 9
		}
		return min(tgo1, int8(3))
	}
}

func point(ok bool) complex64 {
	{
		var tgo2 float32
		if ok {
			return 1
		} else {
			return 2
		}
		return complex(tgo2, float32(3))
	}
}

func pair() (float32, float32) { return 1, 2 }

func main() {
	for _, ok := range []bool{true, false} {
		var tgo3 int8
		if ok {
			tgo3 = 1
		} else {
			tgo3 = 2
		}
		a := min(tgo3, int8(3))
		var tgo4 int8
		if ok {
			tgo4 = 1
		} else {
			tgo4 = 2
		}
		b := max(int8(3), tgo4)
		var tgo5 float32
		if ok {
			tgo5 = 1
		} else {
			tgo5 = 2
		}
		c := complex(tgo5, float32(3))
		var tgo6 float64
		if ok {
			tgo6 = 1
		} else {
			tgo6 = 2
		}
		var tgo7 float64
		if ok {
			tgo7 = 3
		} else {
			tgo7 = 4
		}
		d := complex(tgo6, tgo7)
		var tgo8 float64
		if ok {
			tgo8 = 1.5
		} else {
			tgo8 = 2.5
		}
		e := min(1, tgo8)
		fmt.Printf("%T:%v %T:%v %T:%v %T:%v %T:%v %v %v %v\n", a, a, b, b, c, c, d, d, e, e, choose(ok), point(ok), complex(pair()))
	}
}
