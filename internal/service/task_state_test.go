package service

import (
	"context"
	"path/filepath"
	"testing"

	"krillin-ai/internal/agent"
)

func TestPersistTaskState(t *testing.T) {
	db, err := agent.NewTaskDB(filepath.Join(t.TempDir(), "task_state.db"))
	if err != nil {
		t.Fatalf("NewTaskDB failed: %v", err)
	}
	defer db.Close()

	svc := Service{TaskStateDB: db}
	state := agent.NewTaskState("task-123")
	state.SetStatus(agent.StatusProcessing)
	advanceTaskState(state, agent.StepTranslate)

	if err := svc.persistTaskState(context.Background(), state); err != nil {
		t.Fatalf("persistTaskState failed: %v", err)
	}

	restored, err := db.Get(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("TaskDB.Get failed: %v", err)
	}
	if restored.CurrentStep != agent.StepTranslate {
		t.Fatalf("expected step %s, got %s", agent.StepTranslate, restored.CurrentStep)
	}
	if restored.Status != agent.StatusProcessing {
		t.Fatalf("expected status %s, got %s", agent.StatusProcessing, restored.Status)
	}
}

func TestPersistTaskStateNoDB(t *testing.T) {
	svc := Service{}
	if err := svc.persistTaskState(context.Background(), nil); err != nil {
		t.Fatalf("persistTaskState should be no-op when DB is not configured: %v", err)
	}
}
