package oob

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-multierror"
)

type OOBTester struct {
	MakeFail bool
}

func (o *OOBTester) PowerSet(ctx context.Context, action string) (result string, err error) {
	if o.MakeFail {
		return result, errors.New("power failed")
	}
	return "power action complete: " + action, nil
}

func (o *OOBTester) BootDeviceSet(ctx context.Context, device string, persistent, efiBoot bool) (result string, err error) {
	if o.MakeFail {
		return result, errors.New("boot device failed")
	}
	return "boot device set: " + device, nil
}

func (o *OOBTester) BMCReset(ctx context.Context, rType string) (err error) {
	if o.MakeFail {
		return errors.New("failed: BMC reset")
	}
	return nil
}

func (o *OOBTester) CreateUser(ctx context.Context) (err error) {
	if o.MakeFail {
		return errors.New("create user failed")
	}
	return nil
}

func (o *OOBTester) UpdateUser(ctx context.Context) (err error) {
	if o.MakeFail {
		return errors.New("update user failed")
	}
	return nil
}

func (o *OOBTester) DeleteUser(ctx context.Context) (err error) {
	if o.MakeFail {
		return errors.New("delete user failed")
	}
	return nil
}

func TestMachineBootDevice(t *testing.T) {
	testCases := []struct {
		name       string
		device     string
		makeFail   bool
		err        error
		ctxTimeout time.Duration
	}{
		{name: "success", device: "pxe", err: nil},
		{name: "Power method fails", device: "pxe", makeFail: true, err: &multierror.Error{Errors: []error{errors.New("boot device failed"), errors.New("set boot device failed")}}},
		{name: "error context timeout", device: "pxe", makeFail: true, err: &multierror.Error{Errors: []error{errors.New("context deadline exceeded"), errors.New("set boot device failed")}}, ctxTimeout: 1 * time.Nanosecond},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testImplementation := OOBTester{MakeFail: tc.makeFail}
			expectedResult := "boot device set: " + tc.device
			if tc.ctxTimeout == 0 {
				tc.ctxTimeout = time.Second * 3
			}
			ctx, cancel := context.WithTimeout(context.Background(), tc.ctxTimeout)
			defer cancel()
			result, err := SetBootDevice(ctx, tc.device, true, false, []BootDeviceSetter{&testImplementation})
			if err != nil {
				diff := cmp.Diff(tc.err.Error(), err.Error())
				if diff != "" {
					t.Fatal(diff)
				}

			} else {
				diff := cmp.Diff(expectedResult, result)
				if diff != "" {
					t.Fatal(diff)
				}
			}

		})
	}
}

func TestCreateUser(t *testing.T) {
	testCases := []struct {
		name       string
		makeFail   bool
		err        error
		ctxTimeout time.Duration
	}{
		{name: "success", err: nil},
		{name: "Create User fails", makeFail: true, err: &multierror.Error{Errors: []error{errors.New("create user failed"), errors.New("create user failed")}}},
		{name: "error context timeout", makeFail: true, err: &multierror.Error{Errors: []error{errors.New("context deadline exceeded"), errors.New("create user failed")}}, ctxTimeout: 1 * time.Nanosecond},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testImplementation := OOBTester{MakeFail: tc.makeFail}
			if tc.ctxTimeout == 0 {
				tc.ctxTimeout = time.Second * 3
			}
			ctx, cancel := context.WithTimeout(context.Background(), tc.ctxTimeout)
			defer cancel()
			err := CreateUser(ctx, []BMC{&testImplementation})
			if err != nil {
				diff := cmp.Diff(tc.err.Error(), err.Error())
				if diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	testCases := []struct {
		name       string
		makeFail   bool
		err        error
		ctxTimeout time.Duration
	}{
		{name: "success", err: nil},
		{name: "Update User fails", makeFail: true, err: &multierror.Error{Errors: []error{errors.New("update user failed"), errors.New("update user failed")}}},
		{name: "error context timeout", makeFail: true, err: &multierror.Error{Errors: []error{errors.New("context deadline exceeded"), errors.New("update user failed")}}, ctxTimeout: 1 * time.Nanosecond},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testImplementation := OOBTester{MakeFail: tc.makeFail}
			if tc.ctxTimeout == 0 {
				tc.ctxTimeout = time.Second * 3
			}
			ctx, cancel := context.WithTimeout(context.Background(), tc.ctxTimeout)
			defer cancel()
			err := UpdateUser(ctx, []BMC{&testImplementation})
			if err != nil {
				diff := cmp.Diff(tc.err.Error(), err.Error())
				if diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	testCases := []struct {
		name       string
		makeFail   bool
		err        error
		ctxTimeout time.Duration
	}{
		{name: "success", err: nil},
		{name: "Delete User fails", makeFail: true, err: &multierror.Error{Errors: []error{errors.New("delete user failed"), errors.New("delete user failed")}}},
		{name: "error context timeout", makeFail: true, err: &multierror.Error{Errors: []error{errors.New("context deadline exceeded"), errors.New("delete user failed")}}, ctxTimeout: 1 * time.Nanosecond},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testImplementation := OOBTester{MakeFail: tc.makeFail}
			if tc.ctxTimeout == 0 {
				tc.ctxTimeout = time.Second * 3
			}
			ctx, cancel := context.WithTimeout(context.Background(), tc.ctxTimeout)
			defer cancel()
			err := DeleteUser(ctx, []BMC{&testImplementation})
			if err != nil {
				diff := cmp.Diff(tc.err.Error(), err.Error())
				if diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
