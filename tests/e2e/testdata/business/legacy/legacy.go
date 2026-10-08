package legacy

import (
	"errors"
	"fmt"

	"example.com/business/model"
)

var ErrMissing = errors.New("missing account")

type AccountID string

type AccountStore interface {
	Load(AccountID) (model.Account, error)
}

type MemoryStore struct {
	Accounts map[AccountID]model.Account
}

func (s *MemoryStore) Load(id AccountID) (model.Account, error) {
	account, ok := s.Accounts[id]
	if !ok {
		return model.Account{}, ErrMissing
	}
	return account, nil
}

func Map[T any, U any](values []T, convert func(T) (U, error)) ([]U, error) {
	result := make([]U, 0, len(values))
	for _, value := range values {
		converted, err := convert(value)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

func Stream(values ...model.Account) <-chan model.Account {
	result := make(chan model.Account, len(values))
	for _, value := range values {
		result <- value
	}
	close(result)
	return result
}

func Update(account *model.Account, change func(*model.Account)) {
	change(account)
}

type pointerError struct {
	message string
}

func (e *pointerError) Error() string {
	return e.message
}

func TypedNilError() error {
	var err *pointerError
	return err
}

func Wrap(prefix string, values ...string) string {
	return fmt.Sprintf("%s:%v", prefix, values)
}
