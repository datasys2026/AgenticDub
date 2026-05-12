package agent

import "errors"

type TaskStep string

const (
	StepInit        TaskStep = "init"
	StepSTT        TaskStep = "stt"
	StepTranslate  TaskStep = "translate"
	StepHITLReview TaskStep = "hitl_review"
	StepTTS        TaskStep = "tts"
	StepEmbed      TaskStep = "embed"
	StepDone       TaskStep = "done"
)

type TaskStatus string

const (
	StatusPending       TaskStatus = "pending"
	StatusProcessing    TaskStatus = "processing"
	StatusPaused        TaskStatus = "paused"
	StatusPendingReview TaskStatus = "pending_review"
	StatusApproved      TaskStatus = "approved"
	StatusRejected      TaskStatus = "rejected"
	StatusFailed        TaskStatus = "failed"
	StatusDone          TaskStatus = "done"
)

type TaskState struct {
	TaskID          string
	CurrentStep     TaskStep
	Status          TaskStatus
	History         []TaskStep
	PausedAt        TaskStep
	ReviewApproved  bool
	RejectReason    string
	Error           string
}

func NewTaskState(taskID string) *TaskState {
	return &TaskState{
		TaskID:      taskID,
		CurrentStep: StepInit,
		Status:      StatusPending,
		History:     []TaskStep{StepInit},
	}
}

var ErrInvalidTransition = errors.New("invalid state transition")

var stepOrder = []TaskStep{
	StepInit,
	StepSTT,
	StepTranslate,
	StepHITLReview,
	StepTTS,
	StepEmbed,
	StepDone,
}

func (s *TaskState) CanTransition(to TaskStep) bool {
	currentIdx := -1
	targetIdx := -1

	for i, step := range stepOrder {
		if step == s.CurrentStep {
			currentIdx = i
		}
		if step == to {
			targetIdx = i
		}
	}

	if currentIdx == -1 || targetIdx == -1 {
		return false
	}

	return targetIdx == currentIdx+1
}

func (s *TaskState) Transition(to TaskStep) error {
	if !s.CanTransition(to) {
		return ErrInvalidTransition
	}

	s.CurrentStep = to
	s.History = append(s.History, to)
	return nil
}

func (s *TaskState) SetStatus(status TaskStatus) {
	s.Status = status
}

func (s *TaskState) IsTerminal() bool {
	if s.CurrentStep == StepDone {
		return true
	}
	if s.Status == StatusFailed || s.Status == StatusRejected {
		return true
	}
	return false
}

func (s *TaskState) GetProgress() int {
	switch s.CurrentStep {
	case StepInit:
		return 0
	case StepSTT:
		return 20
	case StepTranslate:
		return 40
	case StepHITLReview:
		return 60
	case StepTTS:
		return 80
	case StepEmbed:
		return 90
	case StepDone:
		return 100
	default:
		return 0
	}
}

func (s *TaskState) Pause() {
	s.Status = StatusPaused
	s.PausedAt = s.CurrentStep
}

func (s *TaskState) Resume() {
	s.Status = StatusProcessing
}

func (s *TaskState) Fail(reason string) {
	s.Status = StatusFailed
	s.Error = reason
}

func (s *TaskState) WaitForHITL() {
	s.Status = StatusPendingReview
	s.PausedAt = StepHITLReview
	s.CurrentStep = StepHITLReview
	s.History = append(s.History, StepHITLReview)
}

func (s *TaskState) ApproveHITL() {
	s.Status = StatusProcessing
}

func (s *TaskState) RejectHITL(reason string) {
	s.Status = StatusRejected
	s.RejectReason = reason
}

func (s *TaskState) Reset() {
	s.CurrentStep = StepInit
	s.Status = StatusPending
	s.History = []TaskStep{StepInit}
	s.PausedAt = ""
	s.ReviewApproved = false
	s.RejectReason = ""
	s.Error = ""
}
