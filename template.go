package template //nolint:revive // package name matches module purpose

import "errors"

// Sentinel errors returned by TemplateProvider implementations.
var (
	// ErrTemplateNotFound is returned by Get when the requested template
	// does not exist.
	ErrTemplateNotFound = errors.New("template: not found")
)

// IsNotFound returns true if the given error is an ErrTemplateNotFound.
// Useful for callers that need to distinguish "not found" from other errors.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrTemplateNotFound)
}
