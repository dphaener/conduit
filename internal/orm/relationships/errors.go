package relationships

import "errors"

var (
	// ErrMaxDepthExceeded is returned when the maximum relationship depth is exceeded
	ErrMaxDepthExceeded = errors.New("maximum relationship depth exceeded")

	// ErrUnknownRelationship is returned when a relationship is not found
	ErrUnknownRelationship = errors.New("unknown relationship")

	// ErrCircularReference is returned when a circular reference is detected
	ErrCircularReference = errors.New("circular reference detected")

	// ErrInvalidRelationType is returned when an invalid relationship type is encountered
	ErrInvalidRelationType = errors.New("invalid relationship type")

	// ErrNoRecords is returned when no records are found
	ErrNoRecords = errors.New("no records found")
)
