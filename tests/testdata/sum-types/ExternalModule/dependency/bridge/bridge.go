package bridge

import "example.com/accounts"

func Personal(name string) accounts.Account {
	return accounts.NewAccountPersonal(name)
}
