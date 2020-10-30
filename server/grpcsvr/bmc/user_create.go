package bmc

import (
	"fmt"
	"strings"

	v1 "github.com/tinkerbell/pbnj/api/v1"
	"github.com/tinkerbell/pbnj/pkg/repository"
)

// CreateUser on a BMC
type CreateUser struct {
	Accessory         Accessory
	CreateUserRequest *v1.CreateUserRequest
}

// userCreateActions is the generic interface for connecting to and creating users
type userCreateActions interface {
	connection
	createUser() (string, repository.Error)
}

type userCreate struct {
	detail connectionDetails
	action userCreateActions
}

type userToModify struct {
	user     string
	password string
	role     string
}

// Create a user on a BMC
func (c *CreateUser) Create() (result string, errMsg repository.Error) {
	host, user, password, errMsg := parseAuth(c.CreateUserRequest.Authn, c.Accessory.StatusMessages, c.Accessory.Log)
	if errMsg.Message != "" {
		return result, errMsg
	}

	userToCreate := userToModify{
		user:     c.CreateUserRequest.GetUserCreds().GetUsername(),
		password: c.CreateUserRequest.GetUserCreds().GetPassword(),
		role:     "Administrator", // TODO put this in the protobuf
	}
	base := "creating user"
	msg := "working on " + base
	sendStatusMessage(msg, c.Accessory.StatusMessages, c.Accessory.Log)

	connections := []userCreate{
		{detail: connectionDetails{name: "bmclib"}, action: &bmclibBMC{user: user, password: password, host: host, userToModify: userToCreate, log: c.Accessory.Log}},
		{detail: connectionDetails{name: "redfish"}, action: &redfishBMC{user: user, password: password, host: host, userToModify: userToCreate, log: c.Accessory.Log}},
	}

	var connected bool
	sendStatusMessage("connecting to BMC", c.Accessory.StatusMessages, c.Accessory.Log)
	for index := range connections {
		connections[index].detail.err = connections[index].action.connect()
		if connections[index].detail.err.Message == "" {
			connections[index].detail.connected = true
			defer connections[index].action.close()
			connected = true
		}
	}
	c.Accessory.Log.V(1).Info("connections", "connections", fmt.Sprintf("%+v", connections))
	if !connected {
		sendStatusMessage("connecting to BMC failed", c.Accessory.StatusMessages, c.Accessory.Log)
		var combinedErrs []string
		for _, connection := range connections {
			combinedErrs = append(combinedErrs, connection.detail.err.Message)
		}
		msg := "could not connect"
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = msg
		errMsg.Details = append(errMsg.Details, combinedErrs...)
		c.Accessory.Log.V(0).Info(msg, "error", combinedErrs)
		return result, errMsg
	}
	sendStatusMessage("connected to BMC", c.Accessory.StatusMessages, c.Accessory.Log)

	for index := range connections {
		if connections[index].detail.connected {
			c.Accessory.Log.V(1).Info(msg, "implementation", connections[index].detail.name)
			result, errMsg = connections[index].action.createUser()
			if errMsg.Message == "" {
				c.Accessory.Log.V(1).Info(base+" succeeded", "implementer", connections[index].detail.name)
				break
			}
		}
	}

	if errMsg.Message != "" {
		sendStatusMessage("error with "+base+": "+errMsg.Message, c.Accessory.StatusMessages, c.Accessory.Log)
		c.Accessory.Log.V(0).Info("error with "+base, "error", errMsg.Message)
	}
	sendStatusMessage(base+" complete", c.Accessory.StatusMessages, c.Accessory.Log)

	return strings.ToLower(result), errMsg //nolint
}
