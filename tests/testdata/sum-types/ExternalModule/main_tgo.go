package main

import accounts_tgo2 "example.com/accounts"

import (
	"example.com/accounts"
	"example.com/accounts/bridge"
	"fmt"
)

func main() {
	var account accounts.Account = bridge.Personal("dependency")
	{
		var value_tgo1 accounts_tgo2.Account = account
		switch value_tgo1.Tag() {
		case accounts.AccountTagPersonal:
			var value_tgo3 accounts.AccountPersonal = value_tgo1.PersonalPayload()
			fmt.Println(value_tgo3.Name)
		default:
			var value_tgo4 accounts.AccountBusiness = value_tgo1.BusinessPayload()
			fmt.Println(value_tgo4.Company)
		}
	}
	var payload_tgo5 accounts.AccountBusiness = accounts.AccountBusiness{Company: "local"}
	var enum_tgo6 accounts.Account = accounts.NewAccountBusiness(payload_tgo5.Company)
	fmt.Println(enum_tgo6.BusinessPayload().Company)
}
