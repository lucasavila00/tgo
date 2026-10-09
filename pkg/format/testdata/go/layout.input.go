//go:build linux
package parity
type record struct {
Short int
Longer int
}
func transform(first int,second int)(int,error){
item:=record{
Short:1, // short
Longer:2, // longer
}
return - -item.Short,nil
}
