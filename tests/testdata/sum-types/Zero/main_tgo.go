package main

import (
	"errors"
	"fmt"
	"unsafe"
)

type Account struct {
	tgoTag      AccountTag
	tgoVariant0 AccountPersonal
	tgoVariant1 AccountBusiness
}
type AccountTag uint8

const (
	AccountTagPersonal AccountTag = 0
	AccountTagBusiness AccountTag = 1
)

func (tgo10 Account) Tag() AccountTag {
	return tgo10.tgoTag
}

type AccountPersonal struct {
	Name    string
	Count   int
	Pointer *int
}

func NewAccountPersonal(tgo11 string, tgo12 int, tgo13 *int) Account {
	return Account{tgoTag: AccountTagPersonal, tgoVariant0: AccountPersonal{tgo11, tgo12, tgo13}}
}
func (tgo14 Account) PersonalPayload() AccountPersonal {
	return tgo14.tgoVariant0
}

type AccountBusiness struct{ Company string }

func NewAccountBusiness(tgo15 string) Account {
	return Account{tgoTag: AccountTagBusiness, tgoVariant1: AccountBusiness{tgo15}}
}
func (tgo16 Account) BusinessPayload() AccountBusiness {
	return tgo16.tgoVariant1
}

type Alias = Account

func zero[T any]() T { var value T; return value }

func failed() (Account, error) { var zero_tgo1 Account; return zero_tgo1, errors.New("failed") }

func label(account Account) string {
	var tgo2 string
	{
		var value_tgo3 Account = account
		switch value_tgo3.Tag() {
		case AccountTagPersonal:
			var v_tgo4 AccountPersonal = value_tgo3.PersonalPayload()
			tgo2 = fmt.Sprintf("personal:%q:%d:%t", v_tgo4.Name, v_tgo4.Count, v_tgo4.Pointer == nil)
		default:
			var v_tgo5 AccountBusiness = value_tgo3.BusinessPayload()
			tgo2 = "business:" + v_tgo5.Company
		}
	}
	return tgo2

}

func main() {
	var account Account
	if (*Account)(unsafe.Pointer(&account)) != &account {
		panic("pointer changed")
	}
	var alias Alias
	closed := make(chan Account)
	close(closed)
	failure, _ := failed()
	values := []Account{
		failure,
		[1]Account{}[0],
		account,
		alias,
		*new(Account),
		make([]Account, 1)[0],
		map[string]Account{}["missing"],
		<-closed,
		zero[Account](),
	}
	for _, value := range values {
		fmt.Println(label(value))
	}
	var payload_tgo6 AccountBusiness = AccountBusiness{Company: "Acme"}
	var enum_tgo7 Account = NewAccountBusiness(payload_tgo6.Company)
	fmt.Println(label(enum_tgo7))
	var payload_tgo8 AccountPersonal = AccountPersonal{Name: "Lucas"}
	var enum_tgo9 Account = NewAccountPersonal(payload_tgo8.Name, payload_tgo8.Count, payload_tgo8.Pointer)
	fmt.Println(label(enum_tgo9))
}
