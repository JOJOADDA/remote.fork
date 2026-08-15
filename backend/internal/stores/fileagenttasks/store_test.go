package fileagenttasks

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/service/agenttask"
)

func TestNewRunsVersionedMigration(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if store == nil {
		t.Fatal("store is nil")
	}
	if _, err := os.Stat(filepath.Join(root, "agent-tasks-schema.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "agent-tasks")); err != nil {
		t.Fatal(err)
	}
}

func TestTaskEventsCheckpointAndRecovery(t *testing.T) {
	ctx := context.Background()
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	task, err := store.Create(ctx, agenttask.CreateInput{ID: "0123456789abcdef", ChatID: "chat-1", ProjectID: "project-1", Provider: "codex", Prompt: "implement feature", Mode: "full-auto"})
	if err != nil {
		t.Fatal(err)
	}
	if task.State != agenttask.StateQueued {
		t.Fatalf("state = %q", task.State)
	}

	payload, _ := json.Marshal(map[string]string{"path": "src/App.tsx"})
	first, err := store.AppendEvent(ctx, task.ID, agenttask.Event{Type: "tool_use", Payload: payload, IdempotencyKey: "task:tool:1"})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := store.AppendEvent(ctx, task.ID, agenttask.Event{Type: "tool_use", Payload: payload, IdempotencyKey: "task:tool:1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Seq != duplicate.Seq {
		t.Fatalf("duplicate seq = %d, first = %d", duplicate.Seq, first.Seq)
	}

	checkpoint, err := store.SaveCheckpoint(ctx, agenttask.Checkpoint{TaskID: task.ID, Phase: "validating", NextAction: "run npm test"})
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.ID == "" {
		t.Fatal("checkpoint id is empty")
	}
	if _, err := store.LatestCheckpoint(ctx, task.ID); err != nil {
		t.Fatal(err)
	}

	updated, err := store.Update(ctx, task.ID, func(task *agenttask.Task) { task.State = agenttask.StateExecuting })
	if err != nil {
		t.Fatal(err)
	}
	if updated.LastEventSeq != first.Seq {
		t.Fatalf("last event seq = %d", updated.LastEventSeq)
	}

	events, err := store.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Seq != 1 {
		t.Fatalf("events = %#v", events)
	}
	recoverable, err := store.ListRecoverable(ctx, time.Now().UTC().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(recoverable) != 1 || recoverable[0].ID != task.ID {
		t.Fatalf("recoverable = %#v", recoverable)
	}
}

func TestLeaseFencingAndRelease(t *testing.T) {
	ctx := context.Background()
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	task, err := store.Create(ctx, agenttask.CreateInput{ID: "fedcba9876543210", ChatID: "chat-2", ProjectID: "project-2", Prompt: "task"})
	if err != nil {
		t.Fatal(err)
	}
	expires := time.Now().UTC().Add(time.Minute)
	if _, err := store.AcquireLease(ctx, task.ID, "worker-a", expires); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquireLease(ctx, task.ID, "worker-b", expires); !errors.Is(err, agenttask.ErrLeaseConflict) {
		t.Fatalf("worker-b acquire error = %v", err)
	}
	if _, err := store.RenewLease(ctx, task.ID, "worker-b", expires.Add(time.Minute)); !errors.Is(err, agenttask.ErrLeaseConflict) {
		t.Fatalf("worker-b renew error = %v", err)
	}
	if err := store.ReleaseLease(ctx, task.ID, "worker-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquireLease(ctx, task.ID, "worker-b", expires); err != nil {
		t.Fatal(err)
	}
}
