package agent

import (
	"context"
	"os"
	"testing"
)

func TestTaskDB_SaveAndGet(t *testing.T) {
	db, err := NewTaskDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	state := NewTaskState("task-123")
	state.CurrentStep = StepSTT
	state.SetStatus(StatusProcessing)

	err = db.Save(context.Background(), state)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	retrieved, err := db.Get(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.TaskID != "task-123" {
		t.Errorf("expected task ID 'task-123', got '%s'", retrieved.TaskID)
	}
	if retrieved.CurrentStep != StepSTT {
		t.Errorf("expected step StepSTT, got '%s'", retrieved.CurrentStep)
	}
}

func TestTaskDB_Get_NotFound(t *testing.T) {
	db, err := NewTaskDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	_, err = db.Get(context.Background(), "non-existent")
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestTaskDB_List(t *testing.T) {
	db, err := NewTaskDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	for i := 1; i <= 3; i++ {
		state := NewTaskState("task-" + string(rune('0'+i)))
		state.CurrentStep = StepSTT
		db.Save(context.Background(), state)
	}

	tasks, err := db.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestTaskDB_Delete(t *testing.T) {
	db, err := NewTaskDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	state := NewTaskState("task-to-delete")
	db.Save(context.Background(), state)

	err = db.Delete(context.Background(), "task-to-delete")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = db.Get(context.Background(), "task-to-delete")
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound after delete, got %v", err)
	}
}

func TestTaskDB_Update(t *testing.T) {
	db, err := NewTaskDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	state := NewTaskState("task-update")
	state.CurrentStep = StepSTT
	db.Save(context.Background(), state)

	state.CurrentStep = StepTranslate
	state.SetStatus(StatusPendingReview)
	err = db.Save(context.Background(), state)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, _ := db.Get(context.Background(), "task-update")
	if retrieved.CurrentStep != StepTranslate {
		t.Errorf("expected StepTranslate after update, got '%s'", retrieved.CurrentStep)
	}
	if retrieved.Status != StatusPendingReview {
		t.Errorf("expected StatusPendingReview after update, got '%s'", retrieved.Status)
	}
}

func TestTaskDB_SaveProgress(t *testing.T) {
	db, err := NewTaskDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	state := NewTaskState("task-progress")
	db.Save(context.Background(), state)

	err = db.SaveProgress(context.Background(), "task-progress", StepTTS, map[string]any{
		"last_processed_segment": 50,
	})
	if err != nil {
		t.Fatalf("SaveProgress failed: %v", err)
	}

	retrieved, _ := db.Get(context.Background(), "task-progress")
	if retrieved.CurrentStep != StepTTS {
		t.Errorf("expected StepTTS, got '%s'", retrieved.CurrentStep)
	}
}

func TestTaskDB_Resume(t *testing.T) {
	db, err := NewTaskDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	state := NewTaskState("task-resume")
	state.CurrentStep = StepTranslate
	state.SetStatus(StatusPaused)
	state.PausedAt = StepTranslate
	db.Save(context.Background(), state)

	resumable, err := db.GetResumable(context.Background())
	if err != nil {
		t.Fatalf("GetResumable failed: %v", err)
	}
	if len(resumable) != 1 {
		t.Errorf("expected 1 resumable task, got %d", len(resumable))
	}
}

func TestTaskDB_ConcurrentSave(t *testing.T) {
	db, err := NewTaskDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			state := NewTaskState("task-" + string(rune('0'+id)))
			state.CurrentStep = StepSTT
			db.Save(context.Background(), state)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	tasks, _ := db.List(context.Background())
	if len(tasks) != 10 {
		t.Errorf("expected 10 tasks, got %d", len(tasks))
	}
}

func TestTaskDB_FileLock(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	db1, err := NewTaskDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create db1: %v", err)
	}
	defer db1.Close()

	db2, err := NewTaskDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create db2: %v", err)
	}
	defer db2.Close()

	_, err = db2.List(context.Background())
	if err != nil {
		t.Errorf("expected no error listing from second connection: %v", err)
	}
}

func TestTaskState_Serializable(t *testing.T) {
	state := NewTaskState("task-serialize")
	state.CurrentStep = StepTTS
	state.SetStatus(StatusProcessing)
	state.Error = "some error"

	data, err := state.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	restored, err := DeserializeTaskState(data)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}
	if restored.TaskID != "task-serialize" {
		t.Errorf("expected task-serialize, got '%s'", restored.TaskID)
	}
	if restored.CurrentStep != StepTTS {
		t.Errorf("expected StepTTS, got '%s'", restored.CurrentStep)
	}
}

func TestTaskDB_ResetTask(t *testing.T) {
	db, err := NewTaskDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	state := NewTaskState("task-reset")
	state.CurrentStep = StepTTS
	state.SetStatus(StatusFailed)
	state.Error = "some error"
	db.Save(context.Background(), state)

	err = db.ResetTask(context.Background(), "task-reset")
	if err != nil {
		t.Fatalf("ResetTask failed: %v", err)
	}

	reset, _ := db.Get(context.Background(), "task-reset")
	if reset.CurrentStep != StepInit {
		t.Errorf("expected StepInit after reset, got '%s'", reset.CurrentStep)
	}
	if reset.Status != StatusPending {
		t.Errorf("expected StatusPending after reset, got '%s'", reset.Status)
	}
}

func TestNewTaskDB_InvalidPath(t *testing.T) {
	_, err := NewTaskDB("/invalid/path/that/does/not/exist/test.db")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestTaskDB_PathCreation(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := tmpDir + "/subdir"
	dbPath := subDir + "/test.db"

	_, err := NewTaskDB(dbPath)
	if err != nil {
		t.Fatalf("NewTaskDB failed: %v", err)
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected db file to be created")
	}
}
