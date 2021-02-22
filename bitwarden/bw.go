package bitwarden

import (
	"errors"
	"fmt"
	"strings"

	"os/exec"

	"github.com/mitchellh/mapstructure"
)

func (c *Client) bwLoginCheck() (bool, error) {
	cmd := exec.Command(c.BitwardenCLIBinary, "login", "--check", "--response")
	isLoggedIn, err := c.runAndCheckSucceeded(cmd, "login --check", 1)
	if err != nil {
		return false, err
	}
	return isLoggedIn, nil
}

func (c *Client) bwUnlockCheck() (bool, error) {
	cmd := exec.Command(c.BitwardenCLIBinary, "unlock", "--check", "--response", "--session", c.SessionKey)
	isUnlocked, err := c.runAndCheckSucceeded(cmd, "unlock --check", 1)
	if err != nil {
		return false, err
	}
	return isUnlocked, nil
}

func (c *Client) bwLogin() error {
	err := c.ensureLoggedOut()
	if err != nil {
		return err
	}
	c.mutexAuth.Lock()
	defer c.mutexAuth.Unlock()
	cmd := exec.Command(c.BitwardenCLIBinary, "login", "--response", c.Email)
	sessionData, err := c.runGivingPasswordExpectingSuccess(cmd, "login", nil)
	if err != nil {
		return err
	}
	var login SessionData
	err = mapstructure.Decode(sessionData, &login)
	if err != nil {
		return err
	}
	c.SessionKey = login.raw
	return nil
}

func (c *Client) bwLogout() error { // TODO should never do?
	//return fmt.Errorf("trying to log out for some reason")
	currentlyLoggedIn, err := c.bwLoginCheck()
	if err != nil {
		return err
	}
	if !currentlyLoggedIn {
		return nil
	}
	c.mutexAuth.Lock()
	defer c.mutexAuth.Unlock()
	cmd := exec.Command(c.BitwardenCLIBinary, "logout", "--response")
	_, err = c.runAndCheckSucceeded(cmd, "logout", 1)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) bwSync() error {
	err := c.ensureUnlocked()
	if err != nil {
		return err
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	cmd := exec.Command(c.BitwardenCLIBinary, "sync", "--response") // NOTE: seems to sometimes ask for password even when giving session token.
	_, err = c.runGivingPasswordExpectingSuccess(cmd, "sync", nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) bwListItems() (*[]interface{}, error) {
	cmd := exec.Command(c.BitwardenCLIBinary, "list", "items", "--response", "--session", c.SessionKey)
	data, err := c.runGivingPasswordExpectingSuccess(cmd, "list items", &map[string]interface{}{
		"data": map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"organizationId": "organization_id",
					"folderId":       "folder_id",
					"card": map[string]interface{}{
						"cardholderName": "cardholder_name",
						"expMonth":       "exp_month",
						"expYear":        "exp_year",
					},
					"identity": map[string]interface{}{
						"firstName":      "first_name",
						"middleName":     "middle_name",
						"lastName":       "last_name",
						"postalCode":     "postal_code",
						"passportNumber": "passport_number",
						"licenseNumber":  "license_number",
					},
					"login": map[string]interface{}{
						"passwordRevisionDate": "password_revision_date",
					},
					"secureNote":    "secure_note",
					"collectionIds": "collection_ids",
					"revisionDate":  "revision_date",
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if list, ok := (*data)["data"].([]interface{}); ok {
		return &list, nil
	} else {
		return nil, fmt.Errorf("unexpected bwListItems output:\n%v", data)
	}
}

func (c *Client) bwLock() error {
	currentlyLoggedIn, err := c.bwLoginCheck()
	if err != nil {
		return err
	}
	if !currentlyLoggedIn {
		return errors.New("cannot lock; not logged in")
	}
	c.mutexAuth.Lock()
	defer c.mutexAuth.Unlock()
	cmd := exec.Command(c.BitwardenCLIBinary, "lock", "--response")
	_, err = c.runAndCheckSucceeded(cmd, "lock", 1)
	if err != nil {
		return err
	}
	return nil
}

type SessionData struct { // NOTE: matches format for both login and unlock.
	noColor bool
	object  string
	title   string
	message string
	raw     string
}

func (c *Client) bwUnlock() error {
	c.mutexAuth.Lock()
	defer c.mutexAuth.Unlock()
	cmd := exec.Command(c.BitwardenCLIBinary, "unlock", "--response")
	sessionData, err := c.runGivingPasswordExpectingSuccess(cmd, "unlock", nil)
	if err != nil {
		return err
	}
	var unlock SessionData
	err = mapstructure.Decode(sessionData, &unlock)
	if err != nil {
		return err
	}
	c.SessionKey = unlock.raw
	return nil
}

func (c *Client) bwVersion() (string, error) {
	cmd := exec.Command(c.BitwardenCLIBinary, "--version")
	version, err := c.runOnly(cmd, "check version", 0)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(*version)), nil
}

type Status struct {
	ServerUrl string
	LastSync  string
	UserEmail string
	UserId    string
	Status    string
}

type StatusOuter struct {
	Object   string
	Template *Status
}

func (c *Client) bwStatus() (*Status, error) {
	cmd := exec.Command(c.BitwardenCLIBinary, "status", "--response", "--session", c.SessionKey)
	var statusOuter StatusOuter
	statusData, err := c.runExpectingSuccess(cmd, "status", nil)
	if err != nil {
		return nil, err
	}
	err = mapstructure.Decode(statusData, &statusOuter)
	if err != nil {
		return nil, err
	}
	return statusOuter.Template, nil
}
