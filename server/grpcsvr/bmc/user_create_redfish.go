package bmc

import (
	"fmt"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
	v1 "github.com/tinkerbell/pbnj/api/v1"
	"github.com/tinkerbell/pbnj/pkg/repository"
)

type redfishBMC struct {
	log          logr.Logger
	conn         *gofish.APIClient
	userToModify userToModify
	user         string
	password     string
	host         string
}

func (r *redfishBMC) connect() repository.Error {
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

func (r *redfishBMC) createUser() (result string, errMsg repository.Error) {
	accounts, err := getAccountList(r.conn)
	if err != nil {
		errMsg.Code = v1.Code_value["UNKNOWN"]
		errMsg.Message = err.Error()
		return "", errMsg
	}

	// BMCs have limited slots for users, getAccountList will return these.
	// also some BMCs like supermicro with redfish v1.0.1 will only show existing
	// users when querying with getAccountList. For those BMCs, we try a POST, as is
	// done down below.
	noRoomforNewUsers := true
	var lastODataID string
	payload := make(map[string]interface{})
	payload["UserName"] = r.userToModify.user
	payload["Password"] = r.userToModify.password
	payload["Enabled"] = true
	payload["RoleId"] = r.userToModify.role

	for _, account := range accounts {
		lastODataID = account.ODataID
		r.log.V(0).Info("account", "account username", account.UserName)
		if len(account.UserName) == 0 && account.ID != "1" { // ID 1 is reserved
			res, err := r.conn.Patch(account.ODataID, payload)
			if err != nil {
				errMsg.Code = v1.Code_value["UNKNOWN"]
				errMsg.Message = err.Error()
				return result, errMsg
			}
			if res.StatusCode != 200 {
				errMsg.Code = v1.Code_value["UNKNOWN"]
				errMsg.Message = fmt.Sprintf("There was an issue with the APIClient. HTTP error code %d", res.StatusCode)
				return result, errMsg
			}
			noRoomforNewUsers = false
			// check to make sure the account was created
			// TODO validate all options are correct
			_, err = getAccount(r.conn, account.ID)
			if err != nil {
				errMsg.Code = v1.Code_value["UNKNOWN"]
				errMsg.Message = fmt.Sprintf("account not found after creating: %v", err.Error())
			}
			// Update Attributes
			r.log.V(0).Info("account", "account", fmt.Sprintf("%+v", account))
			r.log.V(0).Info("ODataID", "ODataID", account.ODataID)
			//oDataID :=
			payload["IpmiLanPrivilege"] = "Administrator"
			payload["IpmiSerialPrivilege"] = "Administrator"
			payload["SolEnable"] = true
			payload["Privilege"] = 511
			break
		}
	}
	if noRoomforNewUsers {
		// one last try
		// this has been shown to work with supermicro with redfish v1.0.1
		var lastTrySuccessful bool
		r.log.V(0).Info("one last try", "lastODataID", lastODataID)
		newODataID := filepath.Dir(lastODataID)
		lastAccountID := filepath.Base(lastODataID)
		if err == nil {
			res, err := r.conn.Post(newODataID, payload)
			if err != nil {
				errMsg.Code = v1.Code_value["UNKNOWN"]
				errMsg.Message = err.Error()
				return result, errMsg
			}
			if res.StatusCode != 201 {
				errMsg.Code = v1.Code_value["UNKNOWN"]
				errMsg.Message = fmt.Sprintf("There was an issue with the APIClient. HTTP error code %d", res.StatusCode)
				return result, errMsg
			}
			_, err = getAccount(r.conn, lastAccountID)
			if err != nil {
				errMsg.Code = v1.Code_value["UNKNOWN"]
				errMsg.Message = fmt.Sprintf("account not found after creating: %v", err.Error())
			}
			lastTrySuccessful = true
		}

		if !lastTrySuccessful {
			errMsg.Code = v1.Code_value["UNKNOWN"]
			errMsg.Message = "There are no room for new users"
		}
	}

	return result, errMsg
}

func getAccountList(c *gofish.APIClient) ([]*redfish.ManagerAccount, error) {
	service := c.Service
	accountService, err := service.AccountService()
	if err != nil {
		return nil, err
	}
	accounts, err := accountService.Accounts()
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func getAccount(c *gofish.APIClient, id string) (*redfish.ManagerAccount, error) {
	accountList, err := getAccountList(c)
	if err != nil {
		return nil, err
	}
	for _, account := range accountList {
		if account.ID == id && len(account.UserName) > 0 {
			return account, nil
		}
	}
	return nil, nil //This will be returned if there was no errors but the user does not exist
}
