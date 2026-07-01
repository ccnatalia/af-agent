package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

const maxSubmitTaskBodyBytes = 1 << 20
const TaskNameDemo = "demo-task"
const TaskNameDownloadFile = "download-file"
const TaskNameMoveFile = "move-file"

type TaskRunner func(payload json.RawMessage) (any, error)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

type SubmitTaskRequest struct {
	RequestID string          `json:"request_id"`
	TaskName  string          `json:"task_name"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type Task struct {
	RequestID  string     `json:"request_id"`
	TaskName   string     `json:"task_name"`
	Status     TaskStatus `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Result     any        `json:"result"`
	Error      *string    `json:"error"`
}

type TaskStore struct {
	mu      sync.RWMutex
	tasks   map[string]*Task
	runners map[string]TaskRunner
}

func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*Task),
		runners: map[string]TaskRunner{
			TaskNameDemo:         executeDemoTask,
			TaskNameDownloadFile: executeDownloadFileTask,
			TaskNameMoveFile:     executeMoveFileTask,
		},
	}
}

func (s *TaskStore) SubmitTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSubmitTaskBodyBytes)

	var req SubmitTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json request body",
		})
		return
	}

	if req.RequestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request_id is required",
		})
		return
	}

	task, created, err := s.getOrCreate(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if created {
		go s.runTask(req)
	}

	writeJSON(w, http.StatusOK, task)
}

func (s *TaskStore) getOrCreate(req SubmitTaskRequest) (Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.tasks[req.RequestID]; ok {
		return cloneTask(task), false, nil
	}

	if req.TaskName == "" {
		return Task{}, false, errors.New("task_name is required")
	}

	if _, ok := s.runners[req.TaskName]; !ok {
		return Task{}, false, errors.New("unknown task_name")
	}

	now := time.Now()
	task := &Task{
		RequestID: req.RequestID,
		TaskName:  req.TaskName,
		Status:    TaskStatusPending,
		CreatedAt: now,
	}
	s.tasks[req.RequestID] = task

	return cloneTask(task), true, nil
}

func (s *TaskStore) runTask(req SubmitTaskRequest) {
	s.markRunning(req.RequestID)

	runner, ok := s.taskRunner(req.TaskName)
	if !ok {
		s.markFailed(req.RequestID, errors.New("unknown task_name"))
		return
	}

	result, err := runner(req.Payload)
	if err != nil {
		s.markFailed(req.RequestID, err)
		return
	}

	s.markSucceeded(req.RequestID, result)
}

func (s *TaskStore) taskRunner(taskName string) (TaskRunner, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runner, ok := s.runners[taskName]
	return runner, ok
}

func (s *TaskStore) markRunning(requestID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := s.tasks[requestID]
	now := time.Now()
	task.Status = TaskStatusRunning
	task.StartedAt = &now
}

func (s *TaskStore) markSucceeded(requestID string, result any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := s.tasks[requestID]
	now := time.Now()
	task.Status = TaskStatusSucceeded
	task.FinishedAt = &now
	task.Result = result
}

func (s *TaskStore) markFailed(requestID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := s.tasks[requestID]
	now := time.Now()
	errMessage := err.Error()
	task.Status = TaskStatusFailed
	task.FinishedAt = &now
	task.Error = &errMessage
}

func executeDemoTask(payload json.RawMessage) (any, error) {
	time.Sleep(5 * time.Second)

	return map[string]any{
		"message":      "task completed",
		"payload_size": len(payload),
	}, nil
}

func cloneTask(task *Task) Task {
	return Task{
		RequestID:  task.RequestID,
		TaskName:   task.TaskName,
		Status:     task.Status,
		CreatedAt:  task.CreatedAt,
		StartedAt:  task.StartedAt,
		FinishedAt: task.FinishedAt,
		Result:     task.Result,
		Error:      task.Error,
	}
}
