package agent

import (
	"testing"
)

func TestTaskState_InitialState(t *testing.T) {
	state := NewTaskState("task-123")

	if state.TaskID != "task-123" {
		t.Errorf("expected task ID 'task-123', got '%s'", state.TaskID)
	}
	if state.CurrentStep != StepInit {
		t.Errorf("expected initial step StepInit, got '%s'", state.CurrentStep)
	}
	if state.Status != StatusPending {
		t.Errorf("expected status StatusPending, got '%s'", state.Status)
	}
}

func TestTaskState_CanTransition(t *testing.T) {
	state := NewTaskState("task-123")

	tests := []struct {
		from    TaskStep
		to      TaskStep
	allowed bool
	}{
		{StepInit, StepSTT, true},
		{StepSTT, StepTranslate, true},
		{StepTranslate, StepHITLReview, true},
		{StepHITLReview, StepTTS, true},
		{StepTTS, StepEmbed, true},
		{StepEmbed, StepDone, true},
		{StepInit, StepDone, false},
		{StepTTS, StepSTT, false},
		{StepTranslate, StepTTS, false},
	}

	for _, tt := range tests {
		state.CurrentStep = tt.from
		result := state.CanTransition(tt.to)
		if result != tt.allowed {
			t.Errorf("CanTransition(%s -> %s): expected %v, got %v", tt.from, tt.to, tt.allowed, result)
		}
	}
}

func TestTaskState_Transition(t *testing.T) {
	state := NewTaskState("task-123")

	err := state.Transition(StepSTT)
	if err != nil {
		t.Fatalf("Transition to StepSTT failed: %v", err)
	}
	if state.CurrentStep != StepSTT {
		t.Errorf("expected current step StepSTT, got '%s'", state.CurrentStep)
	}

	err = state.Transition(StepTranslate)
	if err != nil {
		t.Fatalf("Transition to StepTranslate failed: %v", err)
	}
	if len(state.History) != 3 {
		t.Errorf("expected history length 3, got %d", len(state.History))
	}
}

func TestTaskState_Transition_Invalid(t *testing.T) {
	state := NewTaskState("task-123")

	err := state.Transition(StepDone)
	if err == nil {
		t.Error("expected error when transitioning from StepInit to StepDone")
	}
	if err != ErrInvalidTransition {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestTaskState_SetStatus(t *testing.T) {
	state := NewTaskState("task-123")

	state.SetStatus(StatusProcessing)
	if state.Status != StatusProcessing {
		t.Errorf("expected StatusProcessing, got '%s'", state.Status)
	}

	state.SetStatus(StatusPendingReview)
	if state.Status != StatusPendingReview {
		t.Errorf("expected StatusPendingReview, got '%s'", state.Status)
	}
}

func TestTaskState_IsTerminal(t *testing.T) {
	state := NewTaskState("task-123")

	if state.IsTerminal() {
		t.Error("new task should not be terminal")
	}

	state.CurrentStep = StepDone
	if !state.IsTerminal() {
		t.Error("StepDone should be terminal")
	}

	state.Status = StatusFailed
	state.CurrentStep = StepTranslate
	if !state.IsTerminal() {
		t.Error("StatusFailed should be terminal regardless of step")
	}
}

func TestTaskState_GetProgress(t *testing.T) {
	state := NewTaskState("task-123")

	tests := []struct {
		step    TaskStep
		percent int
	}{
		{StepInit, 0},
		{StepSTT, 20},
		{StepTranslate, 40},
		{StepHITLReview, 60},
		{StepTTS, 80},
		{StepEmbed, 90},
		{StepDone, 100},
	}

	for _, tt := range tests {
		state.CurrentStep = tt.step
		progress := state.GetProgress()
		if progress != tt.percent {
			t.Errorf("Step %s: expected %d%%, got %d%%", tt.step, tt.percent, progress)
		}
	}
}

func TestTaskState_PauseAndResume(t *testing.T) {
	state := NewTaskState("task-123")

	state.Transition(StepSTT)
	state.Transition(StepTranslate)
	state.Pause()

	if state.Status != StatusPaused {
		t.Errorf("expected StatusPaused, got '%s'", state.Status)
	}
	if state.PausedAt != StepTranslate {
		t.Errorf("expected PausedAt StepTranslate, got '%s'", state.PausedAt)
	}

	state.Resume()
	if state.Status != StatusProcessing {
		t.Errorf("expected StatusProcessing after resume, got '%s'", state.Status)
	}
}

func TestTaskState_Failed(t *testing.T) {
	state := NewTaskState("task-123")

	state.Transition(StepSTT)
	state.Fail("STT timeout")

	if state.Status != StatusFailed {
		t.Errorf("expected StatusFailed, got '%s'", state.Status)
	}
	if state.Error != "STT timeout" {
		t.Errorf("expected error 'STT timeout', got '%s'", state.Error)
	}
}

func TestTaskState_HITLWaiting(t *testing.T) {
	state := NewTaskState("task-123")

	state.Transition(StepSTT)
	state.Transition(StepTranslate)
	state.WaitForHITL()

	if state.Status != StatusPendingReview {
		t.Errorf("expected StatusPendingReview, got '%s'", state.Status)
	}
	if state.PausedAt != StepHITLReview {
		t.Errorf("expected PausedAt StepHITLReview, got '%s'", state.PausedAt)
	}
}

func TestTaskState_ApproveHITL(t *testing.T) {
	state := NewTaskState("task-123")

	state.Transition(StepSTT)
	state.Transition(StepTranslate)
	state.WaitForHITL()
	state.ApproveHITL()

	if state.Status != StatusProcessing {
		t.Errorf("expected StatusProcessing after approval, got '%s'", state.Status)
	}
	if state.CurrentStep != StepHITLReview {
		t.Errorf("expected current step StepHITLReview, got '%s'", state.CurrentStep)
	}
	if state.ReviewApproved {
		t.Error("expected ReviewApproved to be false after approval")
	}
}

func TestTaskState_RejectHITL(t *testing.T) {
	state := NewTaskState("task-123")

	state.Transition(StepSTT)
	state.Transition(StepTranslate)
	state.WaitForHITL()
	state.RejectHITL("翻譯品質不佳")

	if state.Status != StatusRejected {
		t.Errorf("expected StatusRejected, got '%s'", state.Status)
	}
	if state.RejectReason != "翻譯品質不佳" {
		t.Errorf("expected reject reason '翻譯品質不佳', got '%s'", state.RejectReason)
	}
}

func TestTaskState_History(t *testing.T) {
	state := NewTaskState("task-123")

	state.Transition(StepSTT)
	state.Transition(StepTranslate)

	if len(state.History) != 3 {
		t.Errorf("expected history length 3, got %d", len(state.History))
	}
	if state.History[0] != StepInit {
		t.Errorf("history[0] expected StepInit, got %s", state.History[0])
	}
	if state.History[1] != StepSTT {
		t.Errorf("history[1] expected StepSTT, got %s", state.History[1])
	}
	if state.History[2] != StepTranslate {
		t.Errorf("history[2] expected StepTranslate, got %s", state.History[2])
	}
}

func TestTaskState_Reset(t *testing.T) {
	state := NewTaskState("task-123")

	state.Transition(StepSTT)
	state.Transition(StepTranslate)
	state.Fail("test")

	state.Reset()

	if state.CurrentStep != StepInit {
		t.Errorf("expected StepInit after reset, got '%s'", state.CurrentStep)
	}
	if state.Status != StatusPending {
		t.Errorf("expected StatusPending after reset, got '%s'", state.Status)
	}
	if state.Error != "" {
		t.Errorf("expected empty error after reset, got '%s'", state.Error)
	}
	if len(state.History) != 1 {
		t.Errorf("expected history length 1 after reset, got %d", len(state.History))
	}
}
