package fileagenttasks

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/service/agenttask"
)

var _ agenttask.Repository = (*Store)(nil)

type Store struct {
	root  string
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func New(root string) (*Store, error) {
	if err := migrate(root); err != nil {
		return nil, err
	}
	return &Store{root: filepath.Join(root, "agent-tasks"), locks: map[string]*sync.Mutex{}}, nil
}

func (s *Store) taskDir(id string) string    { return filepath.Join(s.root, id) }
func (s *Store) taskPath(id string) string   { return filepath.Join(s.taskDir(id), "task.json") }
func (s *Store) eventsPath(id string) string { return filepath.Join(s.taskDir(id), "events.jsonl") }
func (s *Store) checkpointPath(id string) string {
	return filepath.Join(s.taskDir(id), "checkpoint.json")
}

func (s *Store) lock(id string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lock, ok := s.locks[id]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	s.locks[id] = lock
	return lock
}

func newID() string {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:24]
	}
	return hex.EncodeToString(data[:])
}

func validID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'f') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func nowOr(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}

func (s *Store) Create(ctx context.Context, input agenttask.CreateInput) (agenttask.Task, error) {
	if err := ctx.Err(); err != nil {
		return agenttask.Task{}, err
	}
	id := input.ID
	if id == "" {
		id = newID()
	}
	if !validID(id) {
		return agenttask.Task{}, agenttask.ErrInvalidTaskID
	}
	now := time.Now().UTC()
	task := agenttask.Task{ID: id, ChatID: input.ChatID, ProjectID: input.ProjectID, Provider: input.Provider, Model: input.Model, Prompt: input.Prompt, Mode: input.Mode, State: agenttask.StateQueued, Phase: "requested", Acceptance: input.Acceptance, LastActivityAt: now, CreatedAt: now, UpdatedAt: now, RetryLimit: 3}
	lock := s.lock(id)
	lock.Lock()
	defer lock.Unlock()
	if _, err := os.Stat(s.taskPath(id)); err == nil {
		return agenttask.Task{}, os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return agenttask.Task{}, err
	}
	if err := os.MkdirAll(s.taskDir(id), 0o755); err != nil {
		return agenttask.Task{}, err
	}
	if err := writeJSONAtomic(s.taskPath(id), task); err != nil {
		return agenttask.Task{}, err
	}
	file, err := os.OpenFile(s.eventsPath(id), os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		err = file.Close()
	}
	return task, err
}

func (s *Store) Get(ctx context.Context, id string) (agenttask.Task, error) {
	if err := ctx.Err(); err != nil {
		return agenttask.Task{}, err
	}
	if !validID(id) {
		return agenttask.Task{}, agenttask.ErrInvalidTaskID
	}
	data, err := os.ReadFile(s.taskPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return agenttask.Task{}, agenttask.ErrNotFound
	}
	if err != nil {
		return agenttask.Task{}, err
	}
	var task agenttask.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return agenttask.Task{}, err
	}
	return task, nil
}

func (s *Store) readTask(id string) (agenttask.Task, error) {
	data, err := os.ReadFile(s.taskPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return agenttask.Task{}, agenttask.ErrNotFound
	}
	if err != nil {
		return agenttask.Task{}, err
	}
	var task agenttask.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return agenttask.Task{}, err
	}
	return task, nil
}

func (s *Store) Update(ctx context.Context, id string, fn agenttask.UpdateFunc) (agenttask.Task, error) {
	if err := ctx.Err(); err != nil {
		return agenttask.Task{}, err
	}
	lock := s.lock(id)
	lock.Lock()
	defer lock.Unlock()
	task, err := s.Get(ctx, id)
	if err != nil {
		return agenttask.Task{}, err
	}
	fn(&task)
	task.UpdatedAt = time.Now().UTC()
	if task.State.Terminal() && task.CompletedAt == nil {
		completed := task.UpdatedAt
		task.CompletedAt = &completed
	}
	if err := writeJSONAtomic(s.taskPath(id), task); err != nil {
		return agenttask.Task{}, err
	}
	return task, nil
}

func (s *Store) ListRecoverable(ctx context.Context, before time.Time) ([]agenttask.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	tasks := make([]agenttask.Task, 0)
	for _, entry := range entries {
		if !entry.IsDir() || !validID(entry.Name()) {
			continue
		}
		task, err := s.Get(ctx, entry.Name())
		if err != nil {
			continue
		}
		if task.State.Terminal() {
			continue
		}
		if !task.LeaseExpiresAt.IsZero() && task.LeaseExpiresAt.After(before) {
			continue
		}
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.Before(tasks[j].CreatedAt) })
	return tasks, nil
}

func (s *Store) AppendEvent(ctx context.Context, id string, event agenttask.Event) (agenttask.Event, error) {
	if err := ctx.Err(); err != nil {
		return agenttask.Event{}, err
	}
	lock := s.lock(id)
	lock.Lock()
	defer lock.Unlock()
	if _, err := s.Get(ctx, id); err != nil {
		return agenttask.Event{}, err
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.IdempotencyKey == "" {
		event.IdempotencyKey = id + ":" + event.Type + ":" + event.CreatedAt.Format(time.RFC3339Nano)
	}
	events, err := readEvents(s.eventsPath(id))
	if err != nil {
		return agenttask.Event{}, err
	}
	for _, existing := range events {
		if existing.IdempotencyKey == event.IdempotencyKey {
			return existing, nil
		}
	}
	event.Seq = int64(len(events)) + 1
	data, err := json.Marshal(event)
	if err != nil {
		return agenttask.Event{}, err
	}
	data = append(data, '\n')
	file, err := os.OpenFile(s.eventsPath(id), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return agenttask.Event{}, err
	}
	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		return agenttask.Event{}, err
	}
	if err = file.Close(); err != nil {
		return agenttask.Event{}, err
	}
	task, err := s.readTask(id)
	if err != nil {
		return agenttask.Event{}, err
	}
	task.LastEventSeq = event.Seq
	task.LastActivityAt = event.CreatedAt
	task.UpdatedAt = time.Now().UTC()
	if err := writeJSONAtomic(s.taskPath(id), task); err != nil {
		return agenttask.Event{}, err
	}
	return event, nil
}

func (s *Store) EventsAfter(ctx context.Context, id string, after int64) ([]agenttask.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validID(id) {
		return nil, agenttask.ErrInvalidTaskID
	}
	events, err := readEvents(s.eventsPath(id))
	if err != nil {
		return nil, err
	}
	result := make([]agenttask.Event, 0)
	for _, event := range events {
		if event.Seq > after {
			result = append(result, event)
		}
	}
	return result, nil
}

func (s *Store) SaveCheckpoint(ctx context.Context, checkpoint agenttask.Checkpoint) (agenttask.Checkpoint, error) {
	if err := ctx.Err(); err != nil {
		return agenttask.Checkpoint{}, err
	}
	if !validID(checkpoint.TaskID) {
		return agenttask.Checkpoint{}, agenttask.ErrInvalidTaskID
	}
	lock := s.lock(checkpoint.TaskID)
	lock.Lock()
	defer lock.Unlock()
	if _, err := s.Get(ctx, checkpoint.TaskID); err != nil {
		return agenttask.Checkpoint{}, err
	}
	if checkpoint.ID == "" {
		checkpoint.ID = newID()
	}
	if checkpoint.CreatedAt.IsZero() {
		checkpoint.CreatedAt = time.Now().UTC()
	}
	if err := writeJSONAtomic(s.checkpointPath(checkpoint.TaskID), checkpoint); err != nil {
		return agenttask.Checkpoint{}, err
	}
	task, err := s.readTask(checkpoint.TaskID)
	if err != nil {
		return agenttask.Checkpoint{}, err
	}
	task.Phase = checkpoint.Phase
	task.LastActivityAt = checkpoint.CreatedAt
	task.UpdatedAt = time.Now().UTC()
	if err := writeJSONAtomic(s.taskPath(checkpoint.TaskID), task); err != nil {
		return agenttask.Checkpoint{}, err
	}
	return checkpoint, nil
}

func (s *Store) LatestCheckpoint(ctx context.Context, id string) (agenttask.Checkpoint, error) {
	if err := ctx.Err(); err != nil {
		return agenttask.Checkpoint{}, err
	}
	if !validID(id) {
		return agenttask.Checkpoint{}, agenttask.ErrInvalidTaskID
	}
	data, err := os.ReadFile(s.checkpointPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return agenttask.Checkpoint{}, agenttask.ErrNotFound
	}
	if err != nil {
		return agenttask.Checkpoint{}, err
	}
	var checkpoint agenttask.Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return agenttask.Checkpoint{}, err
	}
	return checkpoint, nil
}

func (s *Store) AcquireLease(ctx context.Context, id, owner string, expires time.Time) (agenttask.Task, error) {
	return s.changeLease(ctx, id, owner, expires, false)
}
func (s *Store) RenewLease(ctx context.Context, id, owner string, expires time.Time) (agenttask.Task, error) {
	return s.changeLease(ctx, id, owner, expires, true)
}
func (s *Store) changeLease(ctx context.Context, id, owner string, expires time.Time, renew bool) (agenttask.Task, error) {
	if owner == "" || expires.IsZero() {
		return agenttask.Task{}, agenttask.ErrLeaseConflict
	}
	now := time.Now().UTC()
	var output agenttask.Task
	output, err := s.Update(ctx, id, func(task *agenttask.Task) {
		if renew && task.LeaseOwner != owner {
			return
		}
		if !renew && !task.LeaseExpiresAt.IsZero() && task.LeaseExpiresAt.After(now) && task.LeaseOwner != owner {
			return
		}
		task.LeaseOwner = owner
		task.LeaseExpiresAt = expires
	})
	if err != nil {
		return agenttask.Task{}, err
	}
	if output.LeaseOwner != owner || !output.LeaseExpiresAt.Equal(expires) {
		return agenttask.Task{}, agenttask.ErrLeaseConflict
	}
	return output, nil
}

func (s *Store) ReleaseLease(ctx context.Context, id, owner string) error {
	_, err := s.Update(ctx, id, func(task *agenttask.Task) {
		if task.LeaseOwner == owner {
			task.LeaseOwner = ""
			task.LeaseExpiresAt = time.Time{}
		}
	})
	if err != nil {
		return err
	}
	task, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if task.LeaseOwner != "" {
		return agenttask.ErrLeaseConflict
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func readEvents(path string) ([]agenttask.Event, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	events := make([]agenttask.Event, 0)
	for scanner.Scan() {
		var event agenttask.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}
