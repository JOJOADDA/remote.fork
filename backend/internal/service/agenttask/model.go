package agenttask

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("agent task not found")
	ErrLeaseConflict = errors.New("agent task lease conflict")
	ErrInvalidTaskID = errors.New("invalid agent task id")
)

type State string

const (
	StateQueued          State = "queued"
	StatePlanning        State = "planning"
	StateExecuting       State = "executing"
	StateQuietlyWorking  State = "quietly_working"
	StateStalled         State = "stalled"
	StateReconnecting    State = "reconnecting"
	StateResuming        State = "resuming"
	StateValidating      State = "validating"
	StateReviewing       State = "reviewing"
	StateFixing          State = "fixing"
	StateVerifying       State = "verifying"
	StateCompleted       State = "completed"
	StateBlocked         State = "blocked"
	StateFailed          State = "failed"
	StateCancelRequested State = "cancel_requested"
	StateCancelled       State = "cancelled"
)

func (s State) Terminal() bool {
	return s == StateCompleted || s == StateBlocked || s == StateFailed || s == StateCancelled
}

type Task struct {
	ID             string          `json:"id"`
	ChatID         string          `json:"chatId"`
	ProjectID      string          `json:"projectId"`
	Provider       string          `json:"provider"`
	Model          string          `json:"model,omitempty"`
	Prompt         string          `json:"prompt"`
	Mode           string          `json:"mode"`
	State          State           `json:"state"`
	Phase          string          `json:"phase"`
	Acceptance     json.RawMessage `json:"acceptanceCriteria,omitempty"`
	LastEventSeq   int64           `json:"lastEventSeq"`
	LastActivityAt time.Time       `json:"lastActivityAt"`
	LeaseOwner     string          `json:"leaseOwner,omitempty"`
	LeaseExpiresAt time.Time       `json:"leaseExpiresAt,omitempty"`
	RetryCount     int             `json:"retryCount"`
	RetryLimit     int             `json:"retryLimit"`
	Error          string          `json:"error,omitempty"`
	Result         json.RawMessage `json:"result,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	CompletedAt    *time.Time      `json:"completedAt,omitempty"`
}

type Event struct {
	Seq            int64           `json:"seq"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	IdempotencyKey string          `json:"idempotencyKey"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type Checkpoint struct {
	ID             string          `json:"id"`
	TaskID         string          `json:"taskId"`
	Phase          string          `json:"phase"`
	CompletedSteps []string        `json:"completedSteps,omitempty"`
	RemainingSteps []string        `json:"remainingSteps,omitempty"`
	ChangedFiles   []string        `json:"changedFiles,omitempty"`
	Validation     json.RawMessage `json:"validation,omitempty"`
	NextAction     string          `json:"nextAction"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type CreateInput struct {
	ID         string
	ChatID     string
	ProjectID  string
	Provider   string
	Model      string
	Prompt     string
	Mode       string
	Acceptance json.RawMessage
}

type UpdateFunc func(*Task)

type Repository interface {
	Create(context.Context, CreateInput) (Task, error)
	Get(context.Context, string) (Task, error)
	Update(context.Context, string, UpdateFunc) (Task, error)
	ListRecoverable(context.Context, time.Time) ([]Task, error)
	AppendEvent(context.Context, string, Event) (Event, error)
	EventsAfter(context.Context, string, int64) ([]Event, error)
	SaveCheckpoint(context.Context, Checkpoint) (Checkpoint, error)
	LatestCheckpoint(context.Context, string) (Checkpoint, error)
	AcquireLease(context.Context, string, string, time.Time) (Task, error)
	RenewLease(context.Context, string, string, time.Time) (Task, error)
	ReleaseLease(context.Context, string, string) error
}
