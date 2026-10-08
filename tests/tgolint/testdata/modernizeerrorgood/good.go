package modernizeerrorgood

import "fmt"

func goSource() (*record, error) {
	value, err := load()
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	return value, nil
}
