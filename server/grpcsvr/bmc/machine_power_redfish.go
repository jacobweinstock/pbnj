package bmc

import (
	"time"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
	v1 "github.com/tinkerbell/pbnj/api/v1"
	"github.com/tinkerbell/pbnj/pkg/repository"
)

type redfishBMC struct {
	mAction  MachineAction
	conn     *gofish.APIClient
	user     string
	password string
	host     string
}

func (r *redfishBMC) connection() repository.Error {
	var errMsg repository.Error

	config := gofish.ClientConfig{
		Endpoint: "https://" + r.host,
		Username: r.user,
		Password: r.password,
		Insecure: true,
	}

	c, err := gofish.Connect(config)
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
		return errMsg //nolint
	}
	r.conn = c
	return errMsg
}

func (r *redfishBMC) close() {
	r.conn.Logout()
}

func (r *redfishBMC) on() (result string, errMsg repository.Error) {
	service := r.conn.Service
	ss, err := service.Systems()
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
		return "", errMsg
	}
	for _, system := range ss {
		if system.PowerState == redfish.OnPowerState {
			break
		}
		err = system.Reset(redfish.OnResetType)
		if err != nil {
			errMsg.Code = v1.Code_value["UNKNOWN"]
			errMsg.Message = err.Error()
			return "", errMsg
		}
	}
	return "on", errMsg
}

func (r *redfishBMC) off() (result string, errMsg repository.Error) {
	service := r.conn.Service
	ss, err := service.Systems()
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
		return "", errMsg
	}
	for _, system := range ss {
		if system.PowerState == redfish.OffPowerState {
			break
		}
		err = system.Reset(redfish.GracefulShutdownResetType)
		if err != nil {
			errMsg.Code = v1.Code_value["UNKNOWN"]
			errMsg.Message = err.Error()
			return "", errMsg
		}
	}
	return "off", errMsg
}

func (r *redfishBMC) status() (result string, errMsg repository.Error) {
	service := r.conn.Service
	ss, err := service.Systems()
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
		return "", errMsg
	}
	for _, system := range ss {
		return string(system.PowerState), errMsg
	}
	return result, errMsg
}

func (r *redfishBMC) reset() (result string, errMsg repository.Error) {
	l := r.mAction.Log.GetContextLogger(r.mAction.Ctx)
	service := r.conn.Service
	ss, err := service.Systems()
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
		return "", errMsg
	}
	for _, system := range ss {
		err = system.Reset(redfish.PowerCycleResetType)
		if err != nil {
			l.V(1).Info("warning", "msg", err.Error())
			r.off()
			for wait := 1; wait < 10; wait++ {
				status, _ := r.status()
				if status == "off" {
					break
				}
				time.Sleep(1 * time.Second)
			}
			_, errMsg := r.on()
			return "reset", errMsg
		}
	}
	return "reset", errMsg
}

func (r *redfishBMC) hardoff() (result string, errMsg repository.Error) {
	service := r.conn.Service
	ss, err := service.Systems()
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
		return "", errMsg
	}
	for _, system := range ss {
		if system.PowerState == redfish.OnPowerState {
			break
		}
		err = system.Reset(redfish.ForceOffResetType)
		if err != nil {
			errMsg.Code = v1.Code_value["UNKNOWN"]
			errMsg.Message = err.Error()
			return "", errMsg
		}
	}
	return "hardoff", errMsg
}

func (r *redfishBMC) cycle() (result string, errMsg repository.Error) {
	l := r.mAction.Log.GetContextLogger(r.mAction.Ctx)
	service := r.conn.Service
	ss, err := service.Systems()
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
		return "", errMsg
	}
	for _, system := range ss {
		err = system.Reset(redfish.GracefulRestartResetType)
		if err != nil {
			l.V(1).Info("warning", "msg", err.Error())
			r.off()
			for wait := 1; wait < 10; wait++ {
				status, _ := r.status()
				if status == "off" {
					break
				}
				time.Sleep(1 * time.Second)
			}
			_, errMsg := r.on()
			return "cycle", errMsg
		}
	}
	return "cycle", errMsg
}
