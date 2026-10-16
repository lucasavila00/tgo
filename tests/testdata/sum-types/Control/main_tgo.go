package main

import (
	"errors"
	"fmt"
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

func (tgo39 Account) Tag() AccountTag {
	return tgo39.tgoTag
}

type AccountPersonal struct{ Name string }

func NewAccountPersonal(tgo40 string) Account {
	return Account{tgoTag: AccountTagPersonal, tgoVariant0: AccountPersonal{tgo40}}
}
func (tgo41 Account) PersonalPayload() AccountPersonal {
	return tgo41.tgoVariant0
}

type AccountBusiness struct{ Company string }

func NewAccountBusiness(tgo42 string) Account {
	return Account{tgoTag: AccountTagBusiness, tgoVariant1: AccountBusiness{tgo42}}
}
func (tgo43 Account) BusinessPayload() AccountBusiness {
	return tgo43.tgoVariant1
}

var failure = errors.New("failed")

func read(value string) (int, error) {
	if value == "fail" {
		return 0, failure
	}
	return len(value), nil
}

func number(account Account) (int, error) {
	var zero_tgo6 int
	var tgo1 int
	{
		var value_tgo2 Account = account
		switch value_tgo2.Tag() {
		case AccountTagPersonal:
			var value_tgo3 AccountPersonal = value_tgo2.PersonalPayload()
			if value_tgo3.Name == "return" {
				return -1, nil
			}
			var value_tgo4 int
			var error_tgo5 error
			value_tgo4, error_tgo5 = read(value_tgo3.Name)
			if error_tgo5 != nil {
				return zero_tgo6, error_tgo5
			}
			tgo1 = value_tgo4
		default:
			var value_tgo7 AccountBusiness = value_tgo2.BusinessPayload()
			var value_tgo8 int
			var error_tgo9 error
			value_tgo8, error_tgo9 = read(value_tgo7.Company)
			if error_tgo9 != nil {
				return zero_tgo6, fmt.Errorf("read: %w", error_tgo9)
			}
			tgo1 = value_tgo8
		}
	}
	result := tgo1

	return result, nil
}

func nestedReturn(account Account, active bool) string {
	var tgo10 string
	if active {
		{
			var value_tgo11 Account = account
			switch value_tgo11.Tag() {
			case AccountTagPersonal:
				var value_tgo12 AccountPersonal = value_tgo11.PersonalPayload()
				return "personal:" + value_tgo12.Name
			default:
				var value_tgo13 AccountBusiness = value_tgo11.BusinessPayload()
				return "business:" + value_tgo13.Company
			}
		}

	} else {
		tgo10 = "inactive"
	}
	return tgo10

}

func jumps() {
	var payload_tgo14 AccountPersonal = AccountPersonal{"continue"}
	var enum_tgo15 Account = NewAccountPersonal(payload_tgo14.Name)
	var payload_tgo16 AccountPersonal = AccountPersonal{"break"}
	var enum_tgo17 Account = NewAccountPersonal(payload_tgo16.Name)
	var payload_tgo18 AccountBusiness = AccountBusiness{"ordinary"}
	var enum_tgo19 Account = NewAccountBusiness(payload_tgo18.Company)
	var payload_tgo20 AccountBusiness = AccountBusiness{"goto"}
	var enum_tgo21 Account = NewAccountBusiness(payload_tgo20.Company)
	values := []Account{enum_tgo15, enum_tgo17, enum_tgo19, enum_tgo21}
	var trace []string
outer:
	for _, account := range values {
		{
			var value_tgo22 Account = account
		selected:
			switch value_tgo22.Tag() {
			case AccountTagPersonal:
				var value_tgo23 AccountPersonal = value_tgo22.PersonalPayload()
				if value_tgo23.Name == "continue" {
					continue outer
				}
				trace = append(trace, value_tgo23.Name)
				break selected
			default:
				var value_tgo24 AccountBusiness = value_tgo22.BusinessPayload()
				trace = append(trace, value_tgo24.Company)
				if value_tgo24.Company == "goto" {
					goto done
				}
				break
			}
		}

		trace = append(trace, "tail")
	}
done:
	fmt.Println(trace)
}

func main() {
	var payload_tgo25 AccountPersonal = AccountPersonal{"Ada"}
	var enum_tgo26 Account = NewAccountPersonal(payload_tgo25.Name)
	fmt.Println(nestedReturn(enum_tgo26, true))
	var payload_tgo27 AccountBusiness = AccountBusiness{"Acme"}
	var enum_tgo28 Account = NewAccountBusiness(payload_tgo27.Company)
	fmt.Println(nestedReturn(enum_tgo28, true))
	var payload_tgo29 AccountPersonal = AccountPersonal{"skip"}
	var enum_tgo30 Account = NewAccountPersonal(payload_tgo29.Name)
	fmt.Println(nestedReturn(enum_tgo30, false))
	jumps()
	var payload_tgo31 AccountPersonal = AccountPersonal{"return"}
	var enum_tgo32 Account = NewAccountPersonal(payload_tgo31.Name)
	value, err := number(enum_tgo32)
	fmt.Println(value, err)
	var payload_tgo33 AccountPersonal = AccountPersonal{"fail"}
	var enum_tgo34 Account = NewAccountPersonal(payload_tgo33.Name)
	value, err = number(enum_tgo34)
	fmt.Println(value, err == failure)
	var payload_tgo35 AccountBusiness = AccountBusiness{"fail"}
	var enum_tgo36 Account = NewAccountBusiness(payload_tgo35.Company)
	value, err = number(enum_tgo36)
	fmt.Println(value, err != failure, errors.Is(err, failure))
	var payload_tgo37 AccountBusiness = AccountBusiness{"ok"}
	var enum_tgo38 Account = NewAccountBusiness(payload_tgo37.Company)
	value, err = number(enum_tgo38)
	fmt.Println(value, err)
}
