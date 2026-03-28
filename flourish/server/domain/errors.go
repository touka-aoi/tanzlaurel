package domain

import "errors"

var (
	ErrEntryNotFound = errors.New("entry not found")
	ErrEntryDeleted  = errors.New("entry deleted")
)
