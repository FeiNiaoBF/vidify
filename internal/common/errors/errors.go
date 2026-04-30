// Package errors provides shared error utilities for the Vidify application.
//
// Add domain-agnostic error helpers here as the project grows — for example:
// multi-error aggregation, structured error types for user-facing messages,
// or error classification helpers (transient vs permanent failures).
package errors

import "strings"

// BatchError aggregates multiple errors produced during batch processing.
// It satisfies the standard error interface and the multi-unwrap convention
// introduced in Go 1.20 so callers can use errors.Is / errors.As across
// all wrapped errors.
type BatchError struct {
	errs []error
}

// NewBatchError creates a BatchError from a slice of errors.
// nil entries in errs are silently dropped.
func NewBatchError(errs []error) *BatchError {
	filtered := errs[:0]
	for _, e := range errs {
		if e != nil {
			filtered = append(filtered, e)
		}
	}
	return &BatchError{errs: filtered}
}

// Error returns a single-line summary of all contained errors.
func (b *BatchError) Error() string {
	msgs := make([]string, len(b.errs))
	for i, e := range b.errs {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}

// Unwrap returns all contained errors so errors.Is / errors.As can inspect them.
// The multi-error unwrap convention requires returning []error (Go 1.20+).
func (b *BatchError) Unwrap() []error {
	return b.errs
}

// HasErrors reports whether the batch contains at least one error.
func (b *BatchError) HasErrors() bool {
	return len(b.errs) > 0
}