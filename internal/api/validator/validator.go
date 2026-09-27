package validator

import "errors"

func Required(s string) error {
	if s == "" {
		return errors.New("required")
	}
	return nil
}
