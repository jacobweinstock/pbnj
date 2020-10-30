package bmc

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/tinkerbell/pbnj/api/v1"
	"github.com/tinkerbell/pbnj/pkg/repository"
)

type connectionDetails struct {
	name      string
	connected bool
	err       repository.Error
}

type connection interface {
	connect() repository.Error
	close()
}

// Accessory for all BMC actions
type Accessory struct {
	Log            logr.Logger
	Ctx            context.Context
	StatusMessages chan string
}

func parseAuth(auth *v1.Authn, msgChan chan string, l logr.Logger) (host string, username string, passwd string, errMsg repository.Error) {
	if auth == nil || auth.Authn == nil || auth.GetDirectAuthn() == nil {
		msg := "no auth found"
		sendStatusMessage(msg, msgChan, l)
		errMsg.Code = v1.Code_value["UNAUTHENTICATED"]
		errMsg.Message = msg
		return
	}

	username = auth.GetDirectAuthn().GetUsername()
	passwd = auth.GetDirectAuthn().GetPassword()
	host = auth.GetDirectAuthn().GetHost().GetHost()

	return host, username, passwd, errMsg
}

func sendStatusMessage(msg string, msgChan chan string, l logr.Logger) {
	select {
	case msgChan <- msg:
		return
	case <-time.After(2 * time.Second):
		l.V(0).Info("timed out waiting for status message receiver", "statusMsg", msg)
	}
}
