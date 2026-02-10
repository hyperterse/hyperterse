package logger

import "errors"

// TaggedError carries the logger tag that should be used when the error is
// logged at the top-most boundary.
type TaggedError struct {
	tag string
	err error
}

func (e *TaggedError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *TaggedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Tag returns the associated logger tag.
func (e *TaggedError) Tag() string {
	if e == nil {
		return ""
	}
	return e.tag
}

// WithTag wraps err with a logger tag. If err is nil, nil is returned.
func WithTag(tag string, err error) error {
	if err == nil {
		return nil
	}
	return &TaggedError{
		tag: tag,
		err: err,
	}
}

// ErrorTag extracts a logger tag from an error chain.
func ErrorTag(err error) string {
	if err == nil {
		return ""
	}
	var tagged *TaggedError
	if errors.As(err, &tagged) && tagged != nil {
		return tagged.Tag()
	}
	return ""
}
