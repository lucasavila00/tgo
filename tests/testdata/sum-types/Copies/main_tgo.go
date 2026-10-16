package main

import (
	"fmt"
	"runtime"
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

func (tgo14 Account) Tag() AccountTag {
	return tgo14.tgoTag
}

type AccountPersonal struct {
	Name    string
	Counter *int
}

func NewAccountPersonal(tgo15 string, tgo16 *int) Account {
	return Account{tgoTag: AccountTagPersonal, tgoVariant0: AccountPersonal{tgo15, tgo16}}
}
func (tgo17 Account) PersonalPayload() AccountPersonal {
	return tgo17.tgoVariant0
}

type AccountBusiness struct{ Company string }

func NewAccountBusiness(tgo18 string) Account {
	return Account{tgoTag: AccountTagBusiness, tgoVariant1: AccountBusiness{tgo18}}
}
func (tgo19 Account) BusinessPayload() AccountBusiness {
	return tgo19.tgoVariant1
}

type Archive struct {
	tgoTag     ArchiveTag
	tgoPayload interface {
	}
}
type ArchiveTag uint8

const (
	ArchiveTagEmpty ArchiveTag = 0
	ArchiveTagData  ArchiveTag = 1
)

func (tgo20 Archive) Tag() ArchiveTag {
	return tgo20.tgoTag
}

type ArchiveEmpty struct{}

func NewArchiveEmpty() Archive {
	return Archive{tgoTag: ArchiveTagEmpty}
}
func (tgo21 Archive) EmptyPayload() ArchiveEmpty {
	return ArchiveEmpty{}
}

type ArchiveData struct {
	Values  [128]byte
	Counter *int
}

func NewArchiveData(tgo22 [128]byte, tgo23 *int) Archive {
	return Archive{tgoTag: ArchiveTagData, tgoPayload: ArchiveData{tgo22, tgo23}}
}
func (tgo24 Archive) DataPayload() ArchiveData {
	return tgo24.tgoPayload.(ArchiveData)
}

func archive() Archive {
	count := 11
	var payload_tgo1 ArchiveData = ArchiveData{Values: [128]byte{7}, Counter: &count}
	var enum_tgo2 Archive = NewArchiveData(payload_tgo1.Values, payload_tgo1.Counter)
	return enum_tgo2
}

func boxedCopies() {
	original := archive()
	copied := original
	payload := original.DataPayload()
	var enum_tgo3 Archive = NewArchiveEmpty()
	original = enum_tgo3
	runtime.GC()
	*payload.Counter = 23
	payload.Values[0] = 99
	fmt.Println(copied.DataPayload().Values[0], payload.Values[0], *copied.DataPayload().Counter)
	fmt.Println(original.Tag() == ArchiveTagEmpty)
}

func main() {
	boxedCopies()
	count := 1
	var payload_tgo4 AccountPersonal = AccountPersonal{Name: "Ada", Counter: &count}
	var enum_tgo5 Account = NewAccountPersonal(payload_tgo4.Name, payload_tgo4.Counter)
	account := enum_tgo5
	alias := &account
	var saved func() string
	{
		var value_tgo6 Account = account
		switch value_tgo6.Tag() {
		case AccountTagPersonal:
			var v_tgo7 AccountPersonal = value_tgo6.PersonalPayload()
			var value_tgo8 *Account = alias
			var payload_tgo9 AccountBusiness = AccountBusiness{Company: "Acme"}
			var enum_tgo10 Account = NewAccountBusiness(payload_tgo9.Company)
			*value_tgo8 = enum_tgo10
			*v_tgo7.Counter = 9
			saved = func() string { return v_tgo7.Name }
			fmt.Println(v_tgo7.Name, *v_tgo7.Counter)
		default:
			var v_tgo11 AccountBusiness = value_tgo6.BusinessPayload()
			panic(v_tgo11.Company)
		}
	}

	fmt.Println(saved(), count, account.BusinessPayload().Company)
	var payload_tgo12 AccountPersonal = AccountPersonal{Name: "Grace", Counter: &count}
	var enum_tgo13 Account = NewAccountPersonal(payload_tgo12.Name, payload_tgo12.Counter)
	copy := enum_tgo13
	payload := copy.PersonalPayload()
	payload.Name = "changed"
	fmt.Println(copy.PersonalPayload().Name, payload.Name)
}
