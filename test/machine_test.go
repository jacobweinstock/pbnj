// +build functional

package test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/tinkerbell/pbnj/api/v1"
	v1Client "github.com/tinkerbell/pbnj/client"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
	"github.com/manifoldco/promptui"
)

var (
	PowerRequest_ON      = v1.PowerAction_POWER_ACTION_ON
	PowerRequest_OFF     = v1.PowerAction_POWER_ACTION_OFF
	PowerRequest_STATUS  = v1.PowerAction_POWER_ACTION_STATUS
	PowerRequest_CYCLE   = v1.PowerAction_POWER_ACTION_CYCLE
	PowerRequest_RESET   = v1.PowerAction_POWER_ACTION_RESET
	PowerRequest_HARDOFF = v1.PowerAction_POWER_ACTION_HARDOFF
	DeviceRequest_NONE   = v1.BootDevice_BOOT_DEVICE_NONE
	DeviceRequest_BIOS   = v1.BootDevice_BOOT_DEVICE_BIOS
	DeviceRequest_DISK   = v1.BootDevice_BOOT_DEVICE_DISK
	DeviceRequest_CDROM  = v1.BootDevice_BOOT_DEVICE_CDROM
	DeviceRequest_PXE    = v1.BootDevice_BOOT_DEVICE_PXE
	lookup               = map[string]map[string]expected{
		"happyTests":           happyTests,
		"notIdentifiableTests": notIdentifiableTests,
	}
	happyTests = map[string]expected{
		/*"1 power off": {
			ActionPower: &PowerRequest_OFF,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: OFF",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "off",
				Complete:    true,
				Messages:    []string{"working on power OFF", "connecting to BMC", "connected to BMC", "power OFF complete"},
			},
			WaitTime: 15 * time.Second,
		},
		"2 power status": {
			ActionPower: &PowerRequest_STATUS,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: STATUS",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "off",
				Complete:    true,
				Messages:    []string{"working on power STATUS", "connecting to BMC", "connected to BMC", "power STATUS complete"},
			},
		},*/
		"3 power on": {
			ActionPower: PowerRequest_ON,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: POWER_ACTION_ON",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "on",
				Complete:    true,
				Messages:    []string{"working on power POWER_ACTION_ON", "connected to BMC", "power POWER_ACTION_ON complete"},
			},
			WaitTime: 1 * time.Second,
		},
		"4 power status": {
			ActionPower: PowerRequest_STATUS,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: POWER_ACTION_STATUS",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "on",
				Complete:    true,
				Messages:    []string{"working on power POWER_ACTION_STATUS", "connected to BMC", "power POWER_ACTION_STATUS complete"},
			},
			WaitTime: 1 * time.Second,
		},
		/*"5 power cycle": {
			ActionPower: &PowerRequest_CYCLE,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: CYCLE",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "cycle",
				Complete:    true,
				Messages:    []string{"working on power CYCLE", "connecting to BMC", "connected to BMC", "power CYCLE complete"},
			},
			WaitTime: 60 * time.Second,
		},
		"6 power status": {
			ActionPower: &PowerRequest_STATUS,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: STATUS",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "on",
				Complete:    true,
				Messages:    []string{"working on power STATUS", "connecting to BMC", "connected to BMC", "power STATUS complete"},
			},
		},*/

		//"power hardoff": {Action: &PowerRequest_HARDOFF, Want: notImplementedWant("HARD OFF")},
		//"power reset":   {Action: &PowerRequest_RESET, Want: notImplementedWant("RESET")},
	}
	deviceHappyTests = map[string]expected{
		"1 set device pxe": {
			ActionBootDev: DeviceRequest_PXE,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: OFF",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "off",
				Complete:    true,
				Messages:    []string{"working on power OFF", "connecting to BMC", "connected to BMC", "power OFF complete"},
			},
			WaitTime: 1 * time.Second,
		},
		"2 power status": {
			ActionPower: PowerRequest_STATUS,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: STATUS",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "off",
				Complete:    true,
				Messages:    []string{"working on power STATUS", "connecting to BMC", "connected to BMC", "power STATUS complete"},
			},
		},
		"3 power on": {
			ActionPower: PowerRequest_ON,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: ON",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "on",
				Complete:    true,
				Messages:    []string{"working on power ON", "connecting to BMC", "connected to BMC", "power ON complete"},
			},
			WaitTime: 180 * time.Second,
		},
		"4 power status": {
			ActionPower: PowerRequest_STATUS,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: STATUS",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "on",
				Complete:    true,
				Messages:    []string{"working on power STATUS", "connecting to BMC", "connected to BMC", "power STATUS complete"},
			},
		},
		"5 power cycle": {
			ActionPower: PowerRequest_CYCLE,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: CYCLE",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "cycle",
				Complete:    true,
				Messages:    []string{"working on power CYCLE", "connecting to BMC", "connected to BMC", "power CYCLE complete"},
			},
			WaitTime: 60 * time.Second,
		},
		"6 power status": {
			ActionPower: PowerRequest_STATUS,
			Want: &v1.StatusResponse{
				Id:          "12345",
				Description: "power action: STATUS",
				Error:       &v1.Error{},
				State:       "complete",
				Result:      "on",
				Complete:    true,
				Messages:    []string{"working on power STATUS", "connecting to BMC", "connected to BMC", "power STATUS complete"},
			},
		},
	}
	notIdentifiableTests = map[string]expected{
		"power status":  {ActionPower: PowerRequest_STATUS, Want: notIdentifiableWant},
		"power on":      {ActionPower: PowerRequest_ON, Want: notIdentifiableWant},
		"power off":     {ActionPower: PowerRequest_OFF, Want: notIdentifiableWant},
		"power hardoff": {ActionPower: PowerRequest_HARDOFF, Want: notIdentifiableWant},
		"power cycle":   {ActionPower: PowerRequest_CYCLE, Want: notIdentifiableWant},
		"power reset":   {ActionPower: PowerRequest_RESET, Want: notIdentifiableWant},
	}
	notIdentifiableWant = &v1.StatusResponse{
		Id:          "12345",
		Description: "power action",
		Error: &v1.Error{
			Code:    2,
			Message: "unable to identify the vendor",
			Details: nil,
		},
		State:    "complete",
		Result:   "action failed",
		Complete: true,
		Messages: []string{"connecting to BMC", "connecting to BMC failed"},
	}
)

type expected struct {
	ActionPower   v1.PowerAction
	ActionBootDev v1.BootDevice
	Want          *v1.StatusResponse
	WaitTime      time.Duration
}

type testResource struct {
	Host     string
	Username string
	Password string
	Vendor   string
	Tests    map[string]expected
}

type dataObject map[string]testResource

// TestPower actions against BMCs
func TestPower(t *testing.T) {
	resources := createTestData(cfgData.Data)
	for rname, rs := range resources {
		rs := rs
		rname := rname + "_" + rs.Vendor
		t.Run(rname, func(t *testing.T) {
			t.Parallel()
			tests := rs.Tests
			testsKeys := sortedResources(tests)
			for index, key := range testsKeys {
				key := key
				var failed bool
				tc := tests[key]
				name := key
				t.Run(name, func(t *testing.T) {
					// do the work

					whatAmI := func(i interface{}) (*v1.StatusResponse, error) {
						var got *v1.StatusResponse
						var err error
						switch i.(type) {
						case v1.PowerAction:
							got, err = runMachinePowerClient(rs, tc.ActionPower, cfgData.Server)
							if err != nil {
								return nil, err
							}
						case v1.BootDevice:
							got, err = runMachineBootDevClient(rs, tc.ActionBootDev, cfgData.Server)
							if err != nil {
								return nil, err
							}
						}
						return got, nil
					}

					got, err := whatAmI(tc.ActionPower)
					if err != nil {
						failed = true
						t.Fatal(err)
					}

					got.Id = "12345"
					got.Result = strings.ToLower(got.Result)
					diff := cmp.Diff(tc.Want, got, protocmp.Transform())
					if diff != "" {
						failed = true
						t.Fatalf(diff)
					}
				})
				// user input whether to do next step or stop
				nextStep := tests[testsKeys[index+1]]
				t.Log(nextStep)
				prompt := promptui.Select{
					Label: fmt.Sprintf("next step: %v, Continue?", nextStep.Want.Description),
					Items: []string{"Yes", "No"},
				}

				_, _, err := prompt.Run()
				if err != nil {
					fmt.Printf("Prompt failed %v\n", err)
					return
				}
				if !failed {
					time.Sleep(tc.WaitTime)
				} /*else {
					break
				}*/
			}
		})
	}
}

func runMachinePowerClient(in testResource, action v1.PowerAction, s Server) (*v1.StatusResponse, error) {
	var opts []grpc.DialOption
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(s.URL+":"+s.Port, opts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := v1.NewMachineClient(conn)
	taskClient := v1.NewTaskClient(conn)

	resp, err := v1Client.MachinePower(ctx, client, taskClient, &v1.PowerRequest{
		Authn: &v1.Authn{
			Authn: &v1.Authn_DirectAuthn{
				DirectAuthn: &v1.DirectAuthn{
					Host: &v1.Host{
						Host: in.Host,
					},
					Username: in.Username,
					Password: in.Password,
				},
			},
		},
		Vendor: &v1.Vendor{
			Name: in.Vendor,
		},
		PowerAction: action,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func runMachineBootDevClient(in testResource, action v1.BootDevice, s Server) (*v1.StatusResponse, error) {
	var opts []grpc.DialOption
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(s.URL+":"+s.Port, opts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := v1.NewMachineClient(conn)
	taskClient := v1.NewTaskClient(conn)

	resp, err := v1Client.MachineBootDev(ctx, client, taskClient, &v1.DeviceRequest{
		Authn: &v1.Authn{
			Authn: &v1.Authn_DirectAuthn{
				DirectAuthn: &v1.DirectAuthn{
					Host: &v1.Host{
						Host: in.Host,
					},
					Username: in.Username,
					Password: in.Password,
				},
			},
		},
		Vendor: &v1.Vendor{
			Name: in.Vendor,
		},
		BootDevice: action,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func createTestData(config Data) dataObject {
	f := dataObject{}
	for _, elem := range config.Resources {
		tmp := testResource{}
		tmp.Host = elem.IP
		tmp.Password = elem.Password
		tmp.Username = elem.Username
		tmp.Vendor = elem.Vendor

		tests := map[string]expected{}
		for _, elem := range elem.UseCases.Power {
			t, ok := lookup[elem]
			if !ok {
				fmt.Printf("FYI, useCase: '%v' was not found. please verify it exists in the code base\n", elem)
			}
			for k, v := range t {
				tests[elem+"_"+k] = v
			}
		}
		tmp.Tests = tests

		f[elem.IP] = tmp
	}
	return f
}

func notImplementedWant(fn string) *v1.StatusResponse {
	return &v1.StatusResponse{
		Id:          "12345",
		Description: "power action",
		Error: &v1.Error{
			Code:    12,
			Message: fmt.Sprintf("power %v not implemented", fn),
			Details: nil,
		},
		State:    "complete",
		Result:   "action failed",
		Complete: true,
		Messages: []string{"connecting to BMC", "connected to BMC"},
	}
}

func updateSingleTest(key string, existing map[string]expected, val expected) map[string]expected {
	newExisting := make(map[string]expected)
	for k, v := range existing {
		newExisting[k] = v
	}
	newExisting[key] = val
	return newExisting
}

func sortedResources(m map[string]expected) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}
