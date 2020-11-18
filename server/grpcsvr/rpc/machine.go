package rpc

import (
	"context"

	v1 "github.com/tinkerbell/pbnj/api/v1"
	"github.com/tinkerbell/pbnj/pkg/logging"
	"github.com/tinkerbell/pbnj/pkg/repository"
	"github.com/tinkerbell/pbnj/pkg/task"
	"github.com/tinkerbell/pbnj/server/grpcsvr/oob/machine"
)

// MachineService for doing power and device actions
type MachineService struct {
	Log        logging.Logger
	TaskRunner task.Task
	v1.UnimplementedMachineServer
}

// BootDevice sets the next boot device of a machine
func (m *MachineService) BootDevice(ctx context.Context, in *v1.DeviceRequest) (*v1.DeviceResponse, error) {
	// TODO figure out how not to have to do this, but still keep the logging abstraction clean?
	l := m.Log.GetContextLogger(ctx)
	l.V(0).Info("setting boot device", "device", in.Device.String())

	taskID, err := m.TaskRunner.Execute(
		"setting boot device",
		func(s chan string) (string, repository.Error) {
			mbd, err := machine.NewMachine(
				machine.WithDeviceRequest(in),
				machine.WithLogger(l),
				machine.WithStatusMessage(s),
			)
			if err != nil {
				return "", repository.Error{Message: err.Error()}
			}
			return mbd.BootDevice(ctx)
		})

	return &v1.DeviceResponse{
		TaskId: taskID,
	}, err
}

// Power does a power action against a BMC
func (m *MachineService) Power(ctx context.Context, in *v1.PowerRequest) (*v1.PowerResponse, error) {
	l := m.Log.GetContextLogger(ctx)
	l.V(0).Info("power request")
	// TODO INPUT VALIDATION

	var execFunc = func(s chan string) (string, repository.Error) {
		mp, err := machine.NewMachine(
			machine.WithPowerRequest(in),
			machine.WithLogger(l),
			machine.WithStatusMessage(s),
		)
		if err != nil {
			return "", repository.Error{Message: err.Error()}
		}
		return mp.Power(ctx)
	}
	taskID, err := m.TaskRunner.Execute("power action: "+in.GetAction().String(), execFunc)

	return &v1.PowerResponse{
		TaskId: taskID,
	}, err
}