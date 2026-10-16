package bridge

import "example.com/sumtypes/accounts"

func Personal(name string) accounts.Account {
	return accounts.NewAccountPersonal(name)
}

func Ready() accounts.Flag { return accounts.NewFlagReady() }
