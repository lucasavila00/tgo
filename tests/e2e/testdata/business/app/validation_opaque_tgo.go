package app

import "example.com/business/model"

type validationOpaque struct {
	account model.Account
}

type validationOpaqueNumber struct {
	value int
}
