package errors

import (
	"errors"
	"fmt"
)

var (
	ErrNotImplementedYet = errors.New("not implemented yet")
)

func NotImplementedYetError(what string) error {
	return fmt.Errorf("%s: %w", what, ErrNotImplementedYet)
}
