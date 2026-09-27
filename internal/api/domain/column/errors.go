package column

import "errors"

var (
	ErrHasTasks     = errors.New("column has tasks")
	ErrLastColumn   = errors.New("cannot delete last column")
	ErrNameRequired = errors.New("name is required")
)
