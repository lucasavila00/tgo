package model

import (
	"bytes"
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

type JSONObject struct {
	Seen string `json:"-"`
}

func (JSONObject) MarshalJSON() ([]byte, error) {
	return []byte(`{"custom":"promoted"}`), nil
}

func (value *JSONObject) UnmarshalJSON(data []byte) error {
	value.Seen = string(data)
	return nil
}

func (value JSONInternalPayloadMethodValue) MarshalJSON() ([]byte, error) {
	if value.Seen == "scalar" {
		return []byte(`"scalar"`), nil
	}
	return []byte(`{"custom":"payload"}`), nil
}

func (value *JSONInternalPayloadMethodValue) UnmarshalJSON(data []byte) error {
	value.Seen = string(data)
	return nil
}

func (value *JSONNonNilCustom) UnmarshalJSON(data []byte) error {
	if string(data) == `"valid"` {
		value.Value = &JSONNonNilTarget{}
	}
	return nil
}

func (value *JSONNonNilInternalValue) UnmarshalJSON(data []byte) error {
	if bytes.Contains(data, []byte(`"Required":{}`)) {
		value.Required = &JSONNonNilTarget{}
	}
	return nil
}
