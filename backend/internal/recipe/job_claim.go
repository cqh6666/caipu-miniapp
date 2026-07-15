package recipe

import "errors"

var (
	ErrStaleJobResult          = errors.New("stale recipe job result")
	ErrAutoParseContentChanged = errors.New("recipe content changed while auto-parse was running")
)
