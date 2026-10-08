package model

import (
	"encoding/json"
	"errors"
)

type JSONField string

func (value JSONField) MarshalJSON() ([]byte, error) {
	if value == "bad" {
		return nil, errors.New("field encode error")
	}
	return json.Marshal("custom:" + string(value))
}

func (value *JSONField) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	if text == "bad" {
		return errors.New("field decode error")
	}
	*value = JSONField(text)
	return nil
}
