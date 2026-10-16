package main

import accounts_tgo6 "example.com/sumtypes/accounts"

import (
	model "example.com/sumtypes/accounts"
	"example.com/sumtypes/bridge"
	"fmt"
)

func label(account model.Account) string {
	var tgo4 string
	{
		var value_tgo5 accounts_tgo6.Account = account
		switch value_tgo5.Tag() {
		case model.AccountTagPersonal:
			var value_tgo7 model.AccountPersonal = value_tgo5.PersonalPayload()
			tgo4 = value_tgo7.Name
		default:
			var value_tgo8 model.AccountBusiness = value_tgo5.BusinessPayload()
			tgo4 = value_tgo8.Company
		}
	}
	return tgo4

}
func main() {
	fmt.Println(caseOnly())
	fmt.Println(label(bridge.Personal("Go")))
	var payload_tgo9 model.AccountBusiness = model.AccountBusiness{Company: "TGo"}
	var enum_tgo10 model.Account = model.NewAccountBusiness(payload_tgo9.Company)
	fmt.Println(label(enum_tgo10))
	value := bridge.Personal("API")
	fmt.Println(value.Tag() == model.AccountTagPersonal, value.PersonalPayload().Name)
	var zero model.Account
	fmt.Printf("%q\n", label(zero))
}
