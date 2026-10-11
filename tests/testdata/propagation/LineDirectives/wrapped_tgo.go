//line virtual.tgo:1
package main

import fmt_tgo10 "fmt"

import "fmt"

func wrapped() (tgo8 error) {
	var tgo9 error = tgo8
	{
		var tgo7 error = fail()
		if tgo7 != nil {
			tgo9 = tgo7
			tgo9 = fmt_tgo10.Errorf("%s: %w", "fail", tgo9)
			return tgo9
		}
	}
	fmt.Println("continued wrapped")
	return nil
}
