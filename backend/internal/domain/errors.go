package domain

import "errors"

var (
	ErrNotFound      = errors.New("not_found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("conflict")
	ErrValidation    = errors.New("validation_error")
	ErrInternal      = errors.New("internal_error")
	ErrProviderDown  = errors.New("provider_unavailable")
	ErrAllProviders  = errors.New("all_providers_failed")
)
