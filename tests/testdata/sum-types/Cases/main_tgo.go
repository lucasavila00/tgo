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
	AccountTagClosed   AccountTag = 2
)

func (tgo33 Account) Tag() AccountTag {
	return tgo33.tgoTag
}

type AccountPersonal struct{ Name string }

func NewAccountPersonal(tgo34 string) Account {
	return Account{tgoTag: AccountTagPersonal, tgoVariant0: AccountPersonal{tgo34}}
}
func (tgo35 Account) PersonalPayload() AccountPersonal {
	return tgo35.tgoVariant0
}

type AccountBusiness struct{ Company string }

func NewAccountBusiness(tgo36 string) Account {
	return Account{tgoTag: AccountTagBusiness, tgoVariant1: AccountBusiness{tgo36}}
}
func (tgo37 Account) BusinessPayload() AccountBusiness {
	return tgo37.tgoVariant1
}

type AccountClosed struct{}

func NewAccountClosed() Account {
	return Account{tgoTag: AccountTagClosed}
}
func (tgo38 Account) ClosedPayload() AccountClosed {
	return AccountClosed{}
}

type Alias = Account
type Other Account

func label(value Account) string {
	{
		var value_tgo1 Account = value
		switch value_tgo1.Tag() {
		case AccountTagPersonal:
			var v_tgo2 AccountPersonal = value_tgo1.PersonalPayload()
			return v_tgo2.Name
		case AccountTagBusiness:
			var v_tgo3 AccountBusiness = value_tgo1.BusinessPayload()
			return v_tgo3.Company
		default:
			return "closed"
		}
	}

}

func grouped(value Alias) string {
	{
		var value_tgo4 Alias = value
		switch value_tgo4.Tag() {
		case AccountTagPersonal, AccountTagBusiness:
			var v_tgo5 Alias = value_tgo4
			var same Account = v_tgo5
			return label(same)
		default:
			var v_tgo6 Alias = value_tgo4
			var same Account = v_tgo6
			return label(same)
		}
	}

}

func shadowed() {
	NewAccountPersonal := "constructor"
	AccountPersonal := "payload"
	AccountTagPersonal := "tag"
	var payload_tgo9 AccountPersonal_tgo8 = AccountPersonal_tgo8{Name: "native"}
	var enum_tgo10 Account = NewAccountPersonal_tgo7(payload_tgo9.Name)
	value := enum_tgo10
	var payload_tgo13 AccountPersonal_tgo12 = AccountPersonal_tgo12{Name: "alias"}
	var enum_tgo14 Alias = NewAccountPersonal_tgo11(payload_tgo13.Name)
	alias := enum_tgo14
	{
		var value_tgo15 Account = value
		switch value_tgo15.Tag() {
		case AccountTagPersonal_tgo16:
			var payload_tgo17 AccountPersonal_tgo18 = value_tgo15.PersonalPayload()
			fmt.Println(payload_tgo17.Name, label(alias))
		case AccountTagBusiness:
			panic("wrong business variant")
		default:
			panic("wrong closed variant")
		}
	}

	fmt.Println(NewAccountPersonal, AccountPersonal, AccountTagPersonal)
}

func generic[T Account](value T) string {
	{
		var value_tgo19 T = value
		switch Account(value_tgo19).Tag() {
		case AccountTagPersonal:
			var payload_tgo20 AccountPersonal = Account(value_tgo19).PersonalPayload()
			return payload_tgo20.Name
		case AccountTagBusiness:
			var payload_tgo21 AccountBusiness = Account(value_tgo19).BusinessPayload()
			return payload_tgo21.Company
		default:
			return "closed"
		}
	}

}

func genericLiteral[T Account]() Account {
	var payload_tgo22 AccountPersonal = AccountPersonal{Name: "generic literal"}
	var enum_tgo23 Account = NewAccountPersonal(payload_tgo22.Name)
	return enum_tgo23
}

func main() {
	shadowed()
	var payload_tgo24 AccountPersonal = AccountPersonal{Name: "generic switch"}
	var enum_tgo25 Account = NewAccountPersonal(payload_tgo24.Name)
	fmt.Println(generic(enum_tgo25))
	fmt.Println(label(genericLiteral[Account]()))
	var payload_tgo26 AccountPersonal = AccountPersonal{"Ada"}
	var enum_tgo27 Alias = NewAccountPersonal(payload_tgo26.Name)
	var payload_tgo28 AccountBusiness = AccountBusiness{"Acme"}
	var enum_tgo29 Account = NewAccountBusiness(payload_tgo28.Company)
	var enum_tgo30 Account = NewAccountClosed()
	for _, value := range []Account{enum_tgo27, enum_tgo29, enum_tgo30} {
		fmt.Println(label(value), grouped(value))
	}
	var payload_tgo31 AccountPersonal = AccountPersonal{"Grace"}
	var enum_tgo32 Account = NewAccountPersonal(payload_tgo31.Name)
	other := Other(enum_tgo32)
	fmt.Println(label(Account(other)))
	var ordinary any = "Go"
	switch value := ordinary.(type) {
	case string:
		fmt.Println(value)
	default:
		panic("wrong Go type")
	}
}

var NewAccountPersonal_tgo7 = NewAccountPersonal

type AccountPersonal_tgo8 = AccountPersonal

var NewAccountPersonal_tgo11 = NewAccountPersonal

type AccountPersonal_tgo12 = AccountPersonal

const AccountTagPersonal_tgo16 = AccountTagPersonal

type AccountPersonal_tgo18 = AccountPersonal
