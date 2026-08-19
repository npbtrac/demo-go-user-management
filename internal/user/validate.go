package user

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

func validateCreate(in CreateInput) error {
	if err := validateUsername(in.Username); err != nil {
		return err
	}
	if err := validateEmail(in.Email); err != nil {
		return err
	}
	if err := validatePhone(in.Phone); err != nil {
		return err
	}
	if _, err := normalizeParams(in.Params); err != nil {
		return err
	}
	return nil
}

func validateUsername(username string) error {
	u := strings.TrimSpace(username)
	if u == "" {
		return fmt.Errorf("%w: username is required", ErrInvalid)
	}
	if utf8.RuneCountInString(u) > MaxUsernameLen {
		return fmt.Errorf("%w: username must be at most %d characters", ErrInvalid, MaxUsernameLen)
	}
	if u != username {
		return fmt.Errorf("%w: username must not have leading or trailing whitespace", ErrInvalid)
	}
	return nil
}

func validateEmail(email string) error {
	e := strings.TrimSpace(email)
	if e == "" {
		return fmt.Errorf("%w: email is required", ErrInvalid)
	}
	if utf8.RuneCountInString(e) > MaxEmailLen {
		return fmt.Errorf("%w: email must be at most %d characters", ErrInvalid, MaxEmailLen)
	}
	addr, err := mail.ParseAddress(e)
	if err != nil || addr.Address != e {
		return fmt.Errorf("%w: email is invalid", ErrInvalid)
	}
	return nil
}

func validatePhone(phone *string) error {
	if phone == nil {
		return nil
	}
	p := strings.TrimSpace(*phone)
	if p == "" {
		return nil
	}
	if utf8.RuneCountInString(p) > MaxPhoneLen {
		return fmt.Errorf("%w: phone must be at most %d characters", ErrInvalid, MaxPhoneLen)
	}
	return nil
}

func normalizeParams(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if len(raw) > MaxParamsBytes {
		return nil, fmt.Errorf("%w: params is too large", ErrInvalid)
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" {
		return json.RawMessage(`{}`), nil
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("%w: params must be a JSON object", ErrInvalid)
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("%w: params must be a JSON object", ErrInvalid)
	}
	return out, nil
}

func normalizePhone(phone *string) *string {
	if phone == nil {
		return nil
	}
	p := strings.TrimSpace(*phone)
	if p == "" {
		return nil
	}
	return &p
}
