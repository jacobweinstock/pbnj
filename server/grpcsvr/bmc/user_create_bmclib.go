package bmc

import (
	"github.com/bmc-toolbox/bmclib/cfgresources"
	"github.com/bmc-toolbox/bmclib/devices"
	"github.com/bmc-toolbox/bmclib/discover"
	"github.com/go-logr/logr"
	v1 "github.com/tinkerbell/pbnj/api/v1"
	"github.com/tinkerbell/pbnj/pkg/repository"
)

// bmclib implementation of create for creating a user
type bmclibBMC struct {
	log          logr.Logger
	conn         devices.Bmc
	userToModify userToModify
	user         string
	password     string
	host         string
}

func (b *bmclibBMC) connect() repository.Error {
	var errMsg repository.Error

	connection, err := discover.ScanAndConnect(b.host, b.user, b.password, discover.WithLogger(b.log))
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
		return errMsg //nolint
	}
	switch conn := connection.(type) {
	case devices.Bmc:
		b.conn = conn
	default:
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = "Unknown device"
		return errMsg //nolint
	}
	return errMsg //nolint
}

func (b *bmclibBMC) close() {
	b.conn.Close()
}

func (b *bmclibBMC) createUser() (result string, errMsg repository.Error) {
	var role string
	if b.userToModify.role == "Administrator" {
		role = "admin"
	}
	users := []*cfgresources.User{
		{
			Name:     b.userToModify.user,
			Password: b.userToModify.password,
			Role:     role,
			Enable:   true,
		},
	}
	err := b.conn.User(users)
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
	}
	return result, errMsg
}
