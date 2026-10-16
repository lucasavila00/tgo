package main

import "fmt"

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

func (tgo11 Account) Tag() AccountTag {
	return tgo11.tgoTag
}

type AccountPersonal struct {
	Name    string
	Counter *int
}

func NewAccountPersonal(tgo12 string, tgo13 *int) Account {
	return Account{tgoTag: AccountTagPersonal, tgoVariant0: AccountPersonal{tgo12, tgo13}}
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

func main() {
	count := 1
	var payload_tgo1 AccountPersonal = AccountPersonal{Name: "Ada", Counter: &count}
	var enum_tgo2 Account = NewAccountPersonal(payload_tgo1.Name, payload_tgo1.Counter)
	account := enum_tgo2
	alias := &account
	var saved func() string
	{
		var value_tgo3 Account = account
		switch value_tgo3.Tag() {
		case AccountTagPersonal:
			var v_tgo4 AccountPersonal = value_tgo3.PersonalPayload()
			var value_tgo5 *Account = alias
			var payload_tgo6 AccountBusiness = AccountBusiness{Company: "Acme"}
			var enum_tgo7 Account = NewAccountBusiness(payload_tgo6.Company)
			*value_tgo5 = enum_tgo7
			*v_tgo4.Counter = 9
			saved = func() string { return v_tgo4.Name }
			fmt.Println(v_tgo4.Name, *v_tgo4.Counter)
		default:
			var v_tgo8 AccountBusiness = value_tgo3.BusinessPayload()
			panic(v_tgo8.Company)
		}
	}

	fmt.Println(saved(), count, account.BusinessPayload().Company)
	var payload_tgo9 AccountPersonal = AccountPersonal{Name: "Grace", Counter: &count}
	var enum_tgo10 Account = NewAccountPersonal(payload_tgo9.Name, payload_tgo9.Counter)
	copy := enum_tgo10
	payload := copy.PersonalPayload()
	payload.Name = "changed"
	fmt.Println(copy.PersonalPayload().Name, payload.Name)
}
