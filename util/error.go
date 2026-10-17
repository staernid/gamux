package util

import "errors"

// FixableError is an error interface that carries suggested diagnostic fix actions.
type FixableError interface {
	error
	SuggestedFixes() []string
}

type fixableErr struct {
	err   error
	fixes []string
}

func (e *fixableErr) Error() string {
	return e.err.Error()
}

func (e *fixableErr) Unwrap() error {
	return e.err
}

func (e *fixableErr) SuggestedFixes() []string {
	return e.fixes
}

// WithFixes wraps an error with actionable diagnostic fix suggestions.
func WithFixes(err error, fixes ...string) error {
	if err == nil {
		return nil
	}
	return &fixableErr{
		err:   err,
		fixes: fixes,
	}
}

// ExtractSuggestedFixes returns suggested fixes from a FixableError if present.
func ExtractSuggestedFixes(err error) []string {
	if err == nil {
		return nil
	}
	var fe FixableError
	if errors.As(err, &fe) {
		return fe.SuggestedFixes()
	}
	return nil
}
