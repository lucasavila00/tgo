package foreign
import("fmt";"errors")
var Failure=errors.New("foreign")
var visit,failAt int
type value int
type flag bool
type item struct{Number int}
type key struct{ID int}
type List []item
type Table map[key]item
type Generic[K comparable] map[K]item
func Reset(failure int){visit=0;failAt=failure}
func Number(name string)(value,error){fmt.Print(name);visit++;if visit==failAt{return 0,Failure};return 1,nil}
func Boolean()(flag,error){_,err:=Number("C");return true,err}
func Factory()(func()(value,error),error){_,err:=Number("F");return func()(value,error){return Number("V")},err}
