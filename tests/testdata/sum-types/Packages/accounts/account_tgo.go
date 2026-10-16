package accounts

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

func (tgo1 Account) Tag() AccountTag {
	return tgo1.tgoTag
}

type AccountPersonal struct{ Name string }

func NewAccountPersonal(tgo2 string) Account {
	return Account{tgoTag: AccountTagPersonal, tgoVariant0: AccountPersonal{tgo2}}
}
func (tgo3 Account) PersonalPayload() AccountPersonal {
	return tgo3.tgoVariant0
}

type AccountBusiness struct{ Company string }

func NewAccountBusiness(tgo4 string) Account {
	return Account{tgoTag: AccountTagBusiness, tgoVariant1: AccountBusiness{tgo4}}
}
func (tgo5 Account) BusinessPayload() AccountBusiness {
	return tgo5.tgoVariant1
}

type Flag struct {
	tgoTag FlagTag
}
type FlagTag uint8

const (
	FlagTagReady FlagTag = 0
)

func (tgo6 Flag) Tag() FlagTag {
	return tgo6.tgoTag
}

type FlagReady struct{}

func NewFlagReady() Flag {
	return Flag{tgoTag: FlagTagReady}
}
func (tgo7 Flag) ReadyPayload() FlagReady {
	return FlagReady{}
}
