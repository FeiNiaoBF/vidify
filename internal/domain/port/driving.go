// Package port holds the primary (driving) and secondary (driven) port interfaces
// for Vidify. All layers outside the domain core depend on these interfaces,
// never on concrete infrastructure implementations.
//
// Driving ports are called by adapters (CLI, GUI) to invoke the application.
// Driven ports are called by the application and implemented by infrastructure.
package port

import (
	"context"

	"github.com/FeiNiaoBF/vidify/internal/domain"
)

// Processor is the primary driving port of the Vidify application core.
//
// Driving adapters (CLI today, GUI tomorrow) call through this interface
// instead of depending on the concrete *Service type. This decoupling lets
// adapters be tested with a fake processor and lets the service evolve
// independently of its callers.
type Processor interface {
	// ProcessFile runs the full transcribe → render pipeline for a single audio file.
	ProcessFile(ctx context.Context, cfg domain.Config) Result

	// ProcessBatch runs ProcessFile for every supported audio file in a directory.
	ProcessBatch(ctx context.Context, cfg domain.Config) []Result
}

// Result holds the outcome of processing a single audio file.
// It is defined here so both the application service (which produces it)
// and the driving adapter (which consumes it) can reference it without a
// circular import.
type Result struct {
	// TaskID uniquely identifies this processing task for logging and tracing.
	TaskID string

	// InputPath is the original audio file that was processed.
	InputPath string

	// OutputPath is the path to the generated MP4. Empty on failure.
	OutputPath string

	// Err is non-nil if processing failed at any stage.
	Err error
}
