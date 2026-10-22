package cli

import (
	"context"
	"errors"
	"testing"
)

func TestTaskStateRequiresSequentialPhases(t *testing.T) {
	state := newTaskState()
	if err := state.advance(context.Background(), taskPhaseTranscribe); err == nil {
		t.Fatal("advance() error = nil, want invalid transition error")
	}
	if state.phase != taskPhaseValidate {
		t.Fatalf("phase after invalid transition = %s, want %s", state.phase, taskPhaseValidate)
	}
	if err := state.advance(context.Background(), taskPhaseNormalize); err != nil {
		t.Fatalf("advance() error = %v", err)
	}
}

func TestTaskStateSucceedsAfterPublishing(t *testing.T) {
	state := newTaskState()
	for _, phase := range []taskPhase{
		taskPhaseNormalize,
		taskPhaseTranscribe,
		taskPhaseLayout,
		taskPhaseRender,
		taskPhasePublish,
	} {
		if err := state.advance(context.Background(), phase); err != nil {
			t.Fatalf("advance to %s: %v", phase, err)
		}
	}
	if err := state.finish(nil); err != nil {
		t.Fatalf("finish successful task: %v", err)
	}
	if state.status != taskStatusSucceeded {
		t.Fatalf("terminal status = %s, want %s", state.status, taskStatusSucceeded)
	}
	if err := state.advance(context.Background(), taskPhaseNormalize); err == nil {
		t.Fatal("advance() after success error = nil, want terminal-state error")
	}
	if err := state.finish(nil); err == nil {
		t.Fatal("finish() after success error = nil, want duplicate-terminal error")
	}
}

func TestTaskStateRequiresSuccessOnlyAfterPublishing(t *testing.T) {
	state := newTaskState()
	err := state.finish(nil)
	if err == nil {
		t.Fatal("finish() error = nil, want incomplete-task error")
	}
	var taskErr *taskError
	if !errors.As(err, &taskErr) {
		t.Fatalf("finish() error type = %T, want *taskError", err)
	}
	if taskErr.status != taskStatusFailed {
		t.Errorf("terminal status = %s, want %s", taskErr.status, taskStatusFailed)
	}
}

func TestTaskStateClassifiesFailureAndCancellation(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus taskStatus
	}{
		{name: "failure", err: errors.New("render failed"), wantStatus: taskStatusFailed},
		{name: "cancellation", err: context.Canceled, wantStatus: taskStatusCanceled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := newTaskState()
			if err := state.advance(context.Background(), taskPhaseNormalize); err != nil {
				t.Fatalf("advance() error = %v", err)
			}
			err := state.finish(test.err)
			var taskErr *taskError
			if !errors.As(err, &taskErr) {
				t.Fatalf("finish() error type = %T, want *taskError", err)
			}
			if taskErr.status != test.wantStatus {
				t.Errorf("terminal status = %s, want %s", taskErr.status, test.wantStatus)
			}
			if taskErr.phase != taskPhaseNormalize {
				t.Errorf("failure phase = %s, want %s", taskErr.phase, taskPhaseNormalize)
			}
			if test.wantStatus == taskStatusCanceled && !errors.Is(err, context.Canceled) {
				t.Errorf("finish() error %v does not wrap context.Canceled", err)
			}
		})
	}
}

func TestTaskStateDoesNotAdvanceAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	state := newTaskState()
	if err := state.advance(ctx, taskPhaseNormalize); !errors.Is(err, context.Canceled) {
		t.Fatalf("advance() error = %v, want context.Canceled", err)
	}
	if err := state.finish(context.Canceled); err == nil {
		t.Fatal("finish() error = nil, want canceled task error")
	}
	if state.status != taskStatusCanceled {
		t.Errorf("terminal status = %s, want %s", state.status, taskStatusCanceled)
	}
}
