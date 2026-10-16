package main

import "fmt"

type account struct {
	business bool
	name     string
	count    int
	pointer  *int
	company  string
}

func zero[T any]() T { var value T; return value }
func label(value account) string {
	if value.business {
		return "business:" + value.company
	}
	return fmt.Sprintf("personal:%q:%d:%t", value.name, value.count, value.pointer == nil)
}
func main() {
	var value account
	closed := make(chan account)
	close(closed)
	values := []account{account{}, [1]account{}[0], value, value, *new(account), make([]account, 1)[0], map[string]account{}["missing"], <-closed, zero[account]()}
	for _, value := range values {
		fmt.Println(label(value))
	}
	fmt.Println(label(account{business: true, company: "Acme"}))
	fmt.Println(label(account{name: "Lucas"}))
}
