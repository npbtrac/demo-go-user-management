package user

import "fmt"

func wrapConflict(field string) error {
	return fmt.Errorf("%w: %s already exists", ErrConflict, field)
}
