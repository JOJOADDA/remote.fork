package agenttask

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type Worker func(context.Context, Task) error

type Orchestrator struct {
	repo      Repository
	worker    Worker
	owner     string
	leaseTTL  time.Duration
	heartbeat time.Duration
	poll      time.Duration
	mu        sync.Mutex
	active    map[string]context.CancelFunc
}

type Config struct {
	Owner        string
	LeaseTTL     time.Duration
	Heartbeat    time.Duration
	RecoveryPoll time.Duration
}

func NewOrchestrator(repo Repository, worker Worker, cfg Config) (*Orchestrator, error) {
	if repo == nil {
		return nil, errors.New("agent task repository is required")
	}
	if worker == nil {
		return nil, errors.New("agent task worker is required")
	}
	if cfg.Owner == "" {
		cfg.Owner = fmt.Sprintf("remote-%d", os.Getpid())
	}
	if cfg.LeaseTTL <= 0 {
		cfg.LeaseTTL = 90 * time.Second
	}
	if cfg.Heartbeat <= 0 {
		cfg.Heartbeat = 20 * time.Second
	}
	if cfg.RecoveryPoll <= 0 {
		cfg.RecoveryPoll = 15 * time.Second
	}
	return &Orchestrator{repo: repo, worker: worker, owner: cfg.Owner, leaseTTL: cfg.LeaseTTL, heartbeat: cfg.Heartbeat, poll: cfg.RecoveryPoll, active: map[string]context.CancelFunc{}}, nil
}

func (o *Orchestrator) Start(ctx context.Context) {
	go o.recoveryLoop(ctx)
}

func (o *Orchestrator) Submit(ctx context.Context, input CreateInput) (Task, error) {
	task, err := o.repo.Create(ctx, input)
	if err != nil {
		return Task{}, err
	}
	o.startTask(ctx, task)
	return task, nil
}

func (o *Orchestrator) Cancel(ctx context.Context, id string) error {
	o.mu.Lock()
	cancel := o.active[id]
	o.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	_, err := o.repo.Update(ctx, id, func(task *Task) {
		if !task.State.Terminal() {
			task.State = StateCancelRequested
			task.Phase = "cancelling"
		}
	})
	return err
}

func (o *Orchestrator) recoveryLoop(ctx context.Context) {
	o.recover(ctx)
	ticker := time.NewTicker(o.poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.recover(ctx)
		}
	}
}

func (o *Orchestrator) recover(ctx context.Context) {
	tasks, err := o.repo.ListRecoverable(ctx, time.Now().UTC())
	if err != nil {
		return
	}
	for _, task := range tasks {
		o.startTask(ctx, task)
	}
}

func (o *Orchestrator) startTask(parent context.Context, task Task) {
	o.mu.Lock()
	if _, exists := o.active[task.ID]; exists {
		o.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	o.active[task.ID] = cancel
	o.mu.Unlock()
	go func() {
		defer func() { o.mu.Lock(); delete(o.active, task.ID); o.mu.Unlock() }()
		if _, err := o.repo.AcquireLease(ctx, task.ID, o.owner, time.Now().UTC().Add(o.leaseTTL)); err != nil {
			return
		}
		_, _ = o.repo.Update(ctx, task.ID, func(current *Task) {
			current.State = StateExecuting
			current.Phase = "executing"
			current.LastActivityAt = time.Now().UTC()
		})
		done := make(chan error, 1)
		go func() { done <- o.worker(ctx, task) }()
		heartbeat := time.NewTicker(o.heartbeat)
		defer heartbeat.Stop()
		var runErr error
		running := true
		for running {
			select {
			case runErr = <-done:
				running = false
			case <-heartbeat.C:
				_, err := o.repo.RenewLease(ctx, task.ID, o.owner, time.Now().UTC().Add(o.leaseTTL))
				if err != nil {
					_, _ = o.repo.Update(context.Background(), task.ID, func(current *Task) { current.State = StateStalled; current.Phase = "lease_lost" })

					cancel()
					running = false
				}
				if running {
					_, _ = o.repo.Update(ctx, task.ID, func(current *Task) {
						if !current.State.Terminal() {
							current.State = StateQuietlyWorking
							current.LastActivityAt = time.Now().UTC()
						}
					})
				}
			case <-ctx.Done():
				runErr = ctx.Err()
				running = false
			}
		}
		if runErr == nil {
			_, _ = o.repo.Update(context.Background(), task.ID, func(current *Task) {
				if !current.State.Terminal() {
					current.State = StateCompleted
					current.Phase = "completed"
				}
			})
		} else if errors.Is(runErr, context.Canceled) {
			_, _ = o.repo.Update(context.Background(), task.ID, func(current *Task) {
				current.State = StateCancelled
				current.Phase = "cancelled"
				current.Error = runErr.Error()
			})
		} else {
			_, _ = o.repo.Update(context.Background(), task.ID, func(current *Task) {
				current.State = StateFailed
				current.Phase = "failed"
				current.Error = runErr.Error()
			})
		}
		_ = o.repo.ReleaseLease(context.Background(), task.ID, o.owner)
	}()
}
