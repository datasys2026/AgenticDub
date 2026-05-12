package agent

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskDB struct {
	db  *sql.DB
	mu  sync.Mutex
}

func NewTaskDB(path string) (*TaskDB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path+"?_journal_mode=wal&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	taskDB := &TaskDB{db: db}
	if err := taskDB.init(); err != nil {
		db.Close()
		return nil, err
	}

	return taskDB, nil
}

func (d *TaskDB) init() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			task_id TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			updated_at INTEGER DEFAULT (strftime('%s', 'now'))
		)
	`)
	return err
}

func (d *TaskDB) Close() error {
	return d.db.Close()
}

func (d *TaskDB) Save(ctx context.Context, state *TaskState) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := state.Serialize()
	if err != nil {
		return err
	}

	_, err = d.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO tasks (task_id, data, updated_at) VALUES (?, ?, strftime('%s', 'now'))`,
		state.TaskID, data)
	return err
}

func (d *TaskDB) Get(ctx context.Context, taskID string) (*TaskState, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var data []byte
	err := d.db.QueryRowContext(ctx,
		`SELECT data FROM tasks WHERE task_id = ?`, taskID).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}

	return DeserializeTaskState(data)
}

func (d *TaskDB) Delete(ctx context.Context, taskID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.ExecContext(ctx,
		`DELETE FROM tasks WHERE task_id = ?`, taskID)
	return err
}

func (d *TaskDB) List(ctx context.Context) ([]*TaskState, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.QueryContext(ctx,
		`SELECT data FROM tasks ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*TaskState
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		state, err := DeserializeTaskState(data)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

func (d *TaskDB) SaveProgress(ctx context.Context, taskID string, step TaskStep, progress map[string]any) error {
	var data []byte
	err := d.db.QueryRowContext(ctx,
		`SELECT data FROM tasks WHERE task_id = ?`, taskID).Scan(&data)
	if err == sql.ErrNoRows {
		return ErrTaskNotFound
	}
	if err != nil {
		return err
	}

	state, err := DeserializeTaskState(data)
	if err != nil {
		return err
	}

	state.CurrentStep = step
	state.Status = StatusProcessing

	return d.Save(ctx, state)
}

func (d *TaskDB) GetResumable(ctx context.Context) ([]*TaskState, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.QueryContext(ctx,
		`SELECT data FROM tasks ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*TaskState
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		state, err := DeserializeTaskState(data)
		if err != nil {
			return nil, err
		}
		if state.Status == StatusPaused || state.Status == StatusPendingReview {
			states = append(states, state)
		}
	}
	return states, rows.Err()
}

func (d *TaskDB) ResetTask(ctx context.Context, taskID string) error {
	var data []byte
	err := d.db.QueryRowContext(ctx,
		`SELECT data FROM tasks WHERE task_id = ?`, taskID).Scan(&data)
	if err == sql.ErrNoRows {
		return ErrTaskNotFound
	}
	if err != nil {
		return err
	}

	state, err := DeserializeTaskState(data)
	if err != nil {
		return err
	}

	state.Reset()
	return d.Save(ctx, state)
}
