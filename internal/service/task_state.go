package service

import (
	"context"

	"krillin-ai/internal/agent"
	"krillin-ai/internal/types"
	"krillin-ai/log"

	"go.uber.org/zap"
)

func (s Service) persistTaskState(ctx context.Context, state *agent.TaskState) error {
	if s.TaskStateDB == nil || state == nil {
		return nil
	}
	if err := s.TaskStateDB.Save(ctx, state); err != nil {
		log.GetLogger().Warn("保存 TaskState 到数据库失败", zap.Any("task_id", state.TaskID), zap.Error(err))
		return err
	}
	return nil
}

func syncLegacyStep(taskPtr *types.SubtitleTask, step agent.TaskStep) {
	if taskPtr == nil {
		return
	}
	switch step {
	case agent.StepInit:
		taskPtr.LastSuccessStepNum = 0
	case agent.StepSTT:
		taskPtr.LastSuccessStepNum = 1
	case agent.StepTranslate:
		taskPtr.LastSuccessStepNum = 2
	case agent.StepHITLReview:
		taskPtr.LastSuccessStepNum = 3
	case agent.StepTTS:
		taskPtr.LastSuccessStepNum = 4
	case agent.StepEmbed:
		taskPtr.LastSuccessStepNum = 5
	case agent.StepDone:
		taskPtr.LastSuccessStepNum = 6
	default:
		taskPtr.LastSuccessStepNum = 0
	}
}

func advanceTaskState(state *agent.TaskState, targetStep agent.TaskStep) {
	if state == nil || state.CurrentStep == targetStep {
		return
	}

	if err := state.Transition(targetStep); err != nil {
		state.CurrentStep = targetStep
		state.History = append(state.History, targetStep)
	}
}
