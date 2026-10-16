package main

import accounts_tgo3 "example.com/sumtypes/accounts"

import (
	"example.com/sumtypes/bridge"
)

func caseOnly() string {
	value := bridge.Ready()
	var tgo1 string
	{
		var value_tgo2 accounts_tgo3.Flag = value
		switch value_tgo2.Tag() {
		default:
			tgo1 = "ready"
		}
	}
	return tgo1

}
