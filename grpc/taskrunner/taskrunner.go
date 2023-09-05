package taskrunner

import (
	"context"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"github.com/tinkerbell/pbnj/pkg/metrics"
	"github.com/tinkerbell/pbnj/pkg/repository"
)

// Runner for executing a task.
type Runner struct {
	Repository   repository.Actions
	active       atomic.Int32
	total        atomic.Int32
	orchestrator *orchestrator
}

func NewRunner(repo repository.Actions, maxWorkers int, workerIdleTimeout time.Duration) *Runner {
	o := &orchestrator{
		fifoQueue:      newHostQueue(),
		ingestionQueue: NewIngestQueue(),
		// perIDQueue is a map of hostID to a channel of tasks.
		perIDQueue: sync.Map{},
		manager:    newManager(maxWorkers),
		workerIdleTimeout: workerIdleTimeout,
	}

	return &Runner{
		Repository:   repo,
		orchestrator: o,
	}
}

// ActiveWorkers returns a count of currently active worker jobs.
func (r *Runner) ActiveWorkers() int {
	return int(r.active.Load())
}

// TotalWorkers returns a count total workers executed.
func (r *Runner) TotalWorkers() int {
	return int(r.total.Load())
}

func (r *Runner) Start(ctx context.Context) {
	go r.orchestrator.ingest(ctx)
	go r.orchestrate(ctx)
}

// Execute a task, update repository with status.
func (r *Runner) Execute(_ context.Context, l logr.Logger, description, taskID, host string, action func(chan string) (string, error)) {
	i := Task{
		ID:          taskID,
		Host:        host,
		Description: description,
		Action:      action,
		Log:         l,
	}

	r.orchestrator.ingestionQueue.Enqueue(i)
}

func (r *Runner) updateMessages(ctx context.Context, taskID string, ch chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			record, err := r.Repository.Get(taskID)
			if err != nil {
				return
			}
			record.Messages = append(record.Messages, msg)
			if err := r.Repository.Update(taskID, record); err != nil {
				return
			}
		}
	}
}

// does the work, updates the repo record
// TODO handle retrys, use a timeout.
func (r *Runner) process(ctx context.Context, logger logr.Logger, description, taskID string, action func(chan string) (string, error)) {
	logger = logger.WithValues("taskID", taskID, "description", description)
	r.active.Add(1)
	r.total.Add(1)
	defer func() {
		r.active.Add(-1)
	}()

	metrics.TasksTotal.Inc()
	metrics.TasksActive.Inc()
	defer metrics.TasksActive.Dec()

	messagesChan := make(chan string)
	defer close(messagesChan)
	sessionRecord := repository.Record{
		ID:          taskID,
		Description: description,
		State:       "running",
		Messages:    []string{},
		Error: &repository.Error{
			Code:    0,
			Message: "",
			Details: nil,
		},
	}
	err := r.Repository.Create(taskID, sessionRecord)
	if err != nil {
		return
	}
	cctx, done := context.WithCancel(ctx)
	defer done()
	go r.updateMessages(cctx, taskID, messagesChan)
	logger.Info("worker start")

	resultRecord := repository.Record{
		State:    "complete",
		Complete: true,
		Error: &repository.Error{
			Code:    0,
			Message: "",
			Details: nil,
		},
	}
	result, err := action(messagesChan)
	if err != nil {
		resultRecord.Result = "action failed"
		re, ok := err.(*repository.Error)
		if ok {
			resultRecord.Error = re.StructuredError()
		} else {
			resultRecord.Error.Message = err.Error()
		}
		var foundErr *repository.Error
		if errors.As(err, &foundErr) {
			resultRecord.Error = foundErr.StructuredError()
		}
		logger.Error(err, "task completed with an error")
	}
	record, err := r.Repository.Get(taskID)
	if err != nil {
		return
	}

	record.Complete = resultRecord.Complete
	record.State = resultRecord.State
	record.Result = result
	record.Error = resultRecord.Error

	if err := r.Repository.Update(taskID, record); err != nil {
		logger.Error(err, "failed to update record")
	}

	logger.Info("worker complete", "complete", true)
}

// Status returns the status record of a task.
func (r *Runner) Status(_ context.Context, taskID string) (record repository.Record, err error) {
	record, err = r.Repository.Get(taskID)
	if err != nil {
		switch t := err.(type) {
		case *net.OpError:
			if t.Op == "dial" {
				return record, errors.New("persistence error: unknown host")
			} else if t.Op == "read" {
				return record, errors.New("persistence error: connection refused")
			}
		case syscall.Errno:
			if t == syscall.ECONNREFUSED {
				return record, errors.New("persistence error: connection refused")
			}
		case *url.Error:
			return record, errors.New("persistence error: connection refused")
		}
	}
	return
}
