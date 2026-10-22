package cli

import (
	"context"
	"errors"
	"fmt"
)

type taskPhase uint8

const (
	taskPhaseValidate taskPhase = iota
	taskPhaseNormalize
	taskPhaseTranscribe
	taskPhaseLayout
	taskPhaseRender
	taskPhasePublish
)

func (phase taskPhase) String() string {
	switch phase {
	case taskPhaseValidate:
		return "input validation"
	case taskPhaseNormalize:
		return "audio normalization"
	case taskPhaseTranscribe:
		return "transcription"
	case taskPhaseLayout:
		return "subtitle layout"
	case taskPhaseRender:
		return "video rendering"
	case taskPhasePublish:
		return "output publication"
	default:
		return "unknown phase"
	}
}

type taskStatus uint8

const (
	taskStatusRunning taskStatus = iota
	taskStatusSucceeded
	taskStatusFailed
	taskStatusCanceled
)

func (status taskStatus) String() string {
	switch status {
	case taskStatusRunning:
		return "running"
	case taskStatusSucceeded:
		return "succeeded"
	case taskStatusFailed:
		return "failed"
	case taskStatusCanceled:
		return "canceled"
	default:
		return "unknown status"
	}
}

type taskState struct {
	phase  taskPhase
	status taskStatus
}

func newTaskState() *taskState {
	return &taskState{phase: taskPhaseValidate, status: taskStatusRunning}
}

func (state *taskState) advance(ctx context.Context, next taskPhase) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if state.status != taskStatusRunning {
		return fmt.Errorf("cannot advance a task in %s state", state.status)
	}
	if next != state.phase+1 || next > taskPhasePublish {
		return fmt.Errorf("invalid task phase transition from %s to %s", state.phase, next)
	}
	state.phase = next
	return nil
}

func (state *taskState) finish(err error) error {
	if state.status != taskStatusRunning {
		return fmt.Errorf("task already reached terminal state %s", state.status)
	}
	if err == nil {
		if state.phase != taskPhasePublish {
			err = fmt.Errorf("task completed before output publication (current phase: %s)", state.phase)
		} else {
			state.status = taskStatusSucceeded
			return nil
		}
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		state.status = taskStatusCanceled
	} else {
		state.status = taskStatusFailed
	}
	return &taskError{phase: state.phase, status: state.status, cause: err}
}

type taskError struct {
	phase  taskPhase
	status taskStatus
	cause  error
}

func (err *taskError) Error() string {
	return fmt.Sprintf("task %s during %s: %v", err.status, err.phase, err.cause)
}

func (err *taskError) Unwrap() error {
	return err.cause
}
