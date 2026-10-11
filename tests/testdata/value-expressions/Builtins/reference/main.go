package main

import "fmt"

func main() {
	for _, ok := range []bool{true, false} {
		var a int8
		var c complex64
		var d complex128
		var chosen int8
		var point complex64
		if ok {
			a, c, d, chosen, point = 1, complex(1, 3), complex(1, 3), 7, 1
		} else {
			a, c, d, chosen, point = 2, complex(2, 3), complex(2, 4), 9, 2
		}
		b := int8(3)
		e := float64(1)
		fmt.Printf("%T:%v %T:%v %T:%v %T:%v %T:%v %v %v %v\n", a, a, b, b, c, c, d, d, e, e, chosen, point, complex(float32(1), float32(2)))
	}
}
