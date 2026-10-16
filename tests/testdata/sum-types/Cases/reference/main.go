package main

import "fmt"

type account struct {
	tag  int
	text string
}

func label(value account) string {
	if value.tag == 2 {
		return "closed"
	}
	return value.text
}
func shadowed() {
	NewAccountPersonal := "constructor"
	AccountPersonal := "payload"
	AccountTagPersonal := "tag"
	value := account{text: "native"}
	alias := account{text: "alias"}
	fmt.Println(label(value), label(alias))
	fmt.Println(NewAccountPersonal, AccountPersonal, AccountTagPersonal)
}

func main() {
	shadowed()
	fmt.Println(label(account{text: "generic switch"}))
	fmt.Println(label(account{text: "generic literal"}))
	for _, value := range []account{{text: "Ada"}, {tag: 1, text: "Acme"}, {tag: 2}} {
		fmt.Println(label(value), label(value))
	}
	fmt.Println(label(account{text: "Grace"}))
	var ordinary any = "Go"
	switch value := ordinary.(type) {
	case string:
		fmt.Println(value)
	default:
		panic("wrong Go type")
	}
}
