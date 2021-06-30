package bmc

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/bmc-toolbox/bmclib"
	"github.com/bmc-toolbox/bmclib/bmc"
	"github.com/google/go-cmp/cmp"
	"github.com/packethost/pkg/log/logr"
	v1 "github.com/tinkerbell/pbnj/api/v1"
	"github.com/tinkerbell/pbnj/pkg/repository"
	"github.com/tinkerbell/pbnj/server/grpcsvr/oob"
)

func newAction(withAuthErr bool) Action {
	l, _, _ := logr.NewPacketLogr()
	var authn *v1.Authn_DirectAuthn
	if withAuthErr {
		authn = &v1.Authn_DirectAuthn{
			DirectAuthn: nil,
		}
	} else {
		authn = &v1.Authn_DirectAuthn{
			DirectAuthn: &v1.DirectAuthn{
				Host: &v1.Host{
					Host: "",
				},
				Username: "",
				Password: "",
			},
		}
	}
	m := Action{
		Accessory: oob.Accessory{
			Log:            l,
			StatusMessages: make(chan string),
		},
		ResetBMCRequest: &v1.ResetRequest{
			Authn: &v1.Authn{
				Authn: authn,
			},
			Vendor: &v1.Vendor{
				Name: "local",
			},
			ResetKind: 1,
		},
	}
	return m
}

func TestBMCReset(t *testing.T) {
	m := newAction(false)
	authErr := newAction(true)

	testCases := []struct {
		name         string
		ok           bool
		err          error
		wantErr      error
		resetType    string
		actionStruct Action
	}{
		{"reset err", false, errors.New("bad"), &repository.Error{Code: v1.Code_value["UNKNOWN"], Message: "bad", Details: []string{}}, v1.ResetKind_RESET_KIND_COLD.String(), m},
		{"success", true, nil, nil, v1.ResetKind_RESET_KIND_COLD.String(), m},
		{"reset not ok", false, nil, &repository.Error{Code: v1.Code_value["UNKNOWN"], Message: "reset failed", Details: []string{}}, v1.ResetKind_RESET_KIND_COLD.String(), m},
		{"unknown reset request", true, nil, &repository.Error{Code: v1.Code_value["INVALID_ARGUMENT"], Message: "unknown reset request", Details: []string{}}, "blah", m},
		{"auth parse err", true, nil, &repository.Error{Code: v1.Code_value["UNAUTHENTICATED"], Message: "no auth found", Details: []string{}}, v1.ResetKind_RESET_KIND_COLD.String(), authErr},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			msgs := []string{}
			done := make(chan bool)
			go func() {
			LOOP:
				for {
					select {
					case sm := <-tc.actionStruct.StatusMessages:
						msgs = append(msgs, sm)
					case <-done:
						break LOOP
					}
				}
			}()
			t.Cleanup(func() { done <- true })
			var b *bmclib.Client
			monkey.PatchInstanceMethod(reflect.TypeOf(b), "Open", func(_ *bmclib.Client, _ context.Context) (err error) {
				return nil
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(b), "Close", func(_ *bmclib.Client, _ context.Context) (err error) {
				return nil
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(b), "ResetBMC", func(_ *bmclib.Client, _ context.Context, _ string) (ok bool, err error) {
				return tc.ok, tc.err
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(b), "GetMetadata", func(_ *bmclib.Client) (md bmc.Metadata) {
				return bmc.Metadata{
					SuccessfulProvider:   "redfish",
					ProvidersAttempted:   []string{"ipmitool", "redfish"},
					SuccessfulOpenConns:  []string{"ipmitool", "redfish"},
					SuccessfulCloseConns: []string{},
				}
			})
			err := tc.actionStruct.BMCReset(context.Background(), tc.resetType)
			if err != nil {
				if tc.wantErr != nil {
					if diff := cmp.Diff(err.Error(), tc.wantErr.Error()); diff != "" {
						t.Fatal(diff)
					}
				} else {
					t.Fatal("expected non nil err, got nil")
				}
			} else {
				wantMD := []string{
					"working on bmc reset",
					"{SuccessfulProvider:redfish ProvidersAttempted:[ipmitool redfish] SuccessfulOpenConns:[ipmitool redfish] SuccessfulCloseConns:[]}",
					"bmc reset complete",
				}
				time.Sleep(time.Second)
				if diff := cmp.Diff(msgs, wantMD); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
