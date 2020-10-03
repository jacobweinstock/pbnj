package task

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/xid"
	v1 "github.com/tinkerbell/pbnj/pkg/api/v1"
	"github.com/tinkerbell/pbnj/pkg/logging"
	"github.com/tinkerbell/pbnj/pkg/oob"
	"github.com/tinkerbell/pbnj/pkg/repository"
)

// Task interface for doing BMC actions
type Task interface {
	Execute(ctx context.Context, logger logging.Logger, description string, action func(chan string) (string, *oob.Error)) (id string, err error)
	Status(ctx context.Context, logger logging.Logger, id string) (record repository.Record, err error)
}

// Runner for executing a task
type Runner struct {
	Repository repository.Actions
}

// Execute a task, update repository with status
func (r *Runner) Execute(ctx context.Context, logger logging.Logger, description string, action func(chan string) (string, *oob.Error)) (id string, err error) {
	rawID := xid.New()
	id = rawID.String()
	l := logger.GetContextLogger(ctx)
	l.V(0).Info("executing task", "taskID", id, "taskDescription", description)
	go r.worker(ctx, logger, id, description, action)
	return id, err
}

// does the work, updates the repo record
// TODO handle retrys
func (r *Runner) worker(ctx context.Context, logger logging.Logger, id string, description string, action func(chan string) (string, *oob.Error)) {
	l := logger.GetContextLogger(ctx)
	l.V(0).Info("starting worker", "taskID", id, "description", description)
	resultChan := make(chan string, 1)
	errMsgChan := make(chan oob.Error, 1)
	messagesChan := make(chan string)
	defer close(resultChan)
	defer close(messagesChan)
	repo := r.Repository
	sessionRecord := repository.Record{
		StatusResponse: &v1.StatusResponse{
			Id:          id,
			Description: description,
			State:       "running",
			Messages:    []string{},
			Error: &v1.Error{
				Code:    0,
				Message: "",
				Details: nil,
			},
		}}

	err := repo.Create(ctx, id, sessionRecord)
	if err != nil {
		// TODO how to handle unable to create record; ie network error, persistence error, etc?
		l.V(0).Error(err, "creating record failed")
		return
	}

	go func() {
		for msg := range messagesChan {
			l.V(0).Info("STATUS MESSAGE", "statusMsg", msg)
			currStatus, _ := repo.Get(ctx, id)
			sessionRecord.Messages = append(currStatus.Messages, msg)
			_ = repo.Update(ctx, id, sessionRecord)
		}
	}()

	go func(rChan chan string, sChan chan string, mChan chan oob.Error) {
		result, errMsg := action(messagesChan)
		rChan <- result
		mChan <- *errMsg
	}(resultChan, messagesChan, errMsgChan)

	sessionRecord.Result = <-resultChan
	errMsg := <-errMsgChan // nolint
	// Allow any final status messages to written
	// TODO better way than this??
	time.Sleep(2 * time.Second)
	sessionRecord.State = "complete"
	sessionRecord.Complete = true
	if errMsg.Message != "" {
		l.V(1).Info("error running action", "err", errMsg.Message)
		sessionRecord.Result = "action failed"
		sessionRecord.Error.Code = errMsg.Code
		sessionRecord.Error.Details = errMsg.Details
		sessionRecord.Error.Message = errMsg.Message
	}
	errI := repo.Update(ctx, id, sessionRecord)
	if errI != nil {
		// TODO handle unable to update record; ie network error, persistence error, etc
		l.V(0).Error(err, "updating record failed")
		return
	}
}

// Status returns the status record of a task
func (r *Runner) Status(ctx context.Context, logger logging.Logger, id string) (record repository.Record, err error) {
	l := logger.GetContextLogger(ctx)
	l.V(0).Info("getting task record", "taskID", id)
	record, err = r.Repository.Get(ctx, id)
	if err != nil {
		l.V(0).Error(err, "error getting task")
		l.V(0).Info(fmt.Sprintf("%T", err))
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