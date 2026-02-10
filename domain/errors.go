package domain

import "errors"

var (
	// ErrUnauthorized indicates missing or invalid credentials.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrRantTooLong indicates the rant exceeds the character limit.
	ErrRantTooLong = errors.New("rant exceeds character limit")

	// ErrEmptyRant indicates the user submitted an empty rant.
	ErrEmptyRant = errors.New("rant cannot be empty")
)
