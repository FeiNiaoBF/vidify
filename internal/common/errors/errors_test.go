package errors_test

import (
	"errors"
	"fmt"
	"testing"

	customerrors "github.com/FeiNiaoBF/vidify/internal/common/errors"
)

func TestBatchError_Error(t *testing.T) {
	t.Parallel()

	e1 := fmt.Errorf("first error")
	e2 := fmt.Errorf("second error")
	batch := customerrors.NewBatchError([]error{e1, e2})

	got := batch.Error()
	if got != "first error; second error" {
		t.Errorf("Error() = %q, want %q", got, "first error; second error")
	}
}

func TestBatchError_Unwrap(t *testing.T) {
	t.Parallel()

	sentinel := fmt.Errorf("sentinel")
	batch := customerrors.NewBatchError([]error{fmt.Errorf("wrap: %w", sentinel)})

	if !errors.Is(batch, sentinel) {
		t.Error("errors.Is should find the sentinel through BatchError.Unwrap")
	}
}

func TestBatchError_NilsDropped(t *testing.T) {
	t.Parallel()

	batch := customerrors.NewBatchError([]error{nil, fmt.Errorf("only this"), nil})
	if batch.Error() != "only this" {
		t.Errorf("Error() = %q, want %q", batch.Error(), "only this")
	}
}

func TestBatchError_HasErrors(t *testing.T) {
	t.Parallel()

	empty := customerrors.NewBatchError(nil)
	if empty.HasErrors() {
		t.Error("HasErrors() should be false for an empty batch")
	}

	nonil := customerrors.NewBatchError([]error{nil})
	if nonil.HasErrors() {
		t.Error("HasErrors() should be false when all entries are nil")
	}

	nonEmpty := customerrors.NewBatchError([]error{fmt.Errorf("e")})
	if !nonEmpty.HasErrors() {
		t.Error("HasErrors() should be true when at least one error is present")
	}
}
