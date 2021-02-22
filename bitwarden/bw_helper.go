package bitwarden

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"io"
	"log"
	"os/exec"
	"strings"
)

// use env stuff to reset local env.
// where is login persisted? can swapping envs around allow the same binary to be used for multiple sessions simultaneously?

type Response struct {
	Success bool                   `json:"success,omitempty"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func UnmarshalConvertKeys(data *[]byte, keyConversion *map[string]interface{}) (*map[string]interface{}, error) {
	// NOTE: keyConversion is almost json compatible. leaf values are all strings, which are the name to rename the keys.
	// todo name conversion should be recursive function given pointer to the relevant portion of the map and the relevant portion of the keyConversion.
	// handle json data with list as root; likely by wrapping.
	// todo handle renaming keys when you also want to rename the maps of those keys etc
	if !json.Valid(*data) {
		return nil, fmt.Errorf("invalid json:\n%s", string(*data))
	}
	var generic map[string]interface{}
	err := json.Unmarshal(*data, &generic)
	if err != nil {
		return nil, err
	}
	err = convertKeys(&generic, keyConversion)
	if err != nil {
		return nil, err
	}

	return &generic, nil
}

func convertKeys(input *map[string]interface{}, keyConversion *map[string]interface{}) error {
	// careful: https://stackoverflow.com/questions/45132563/idiomatic-way-of-renaming-keys-in-map-while-ranging-over-the-original-map
	// TODO handle lists in parallel? this could be a bottleneck.
	// NOTE: this will run into issues with lists of items that have different structures; only some of them containing maps etc.
	// NOTE: fails on nested lists.
	if keyConversion == nil {
		return nil
	}
	for key, element := range *keyConversion {
		switch value := element.(type) {
		case string:
			convertKey(input, key, value)
		case []map[string]interface{}:
			// TODO check that keyConversion list has a single element? or let it help differentiate list items with differing structure.
			switch inputChild := (*input)[key].(type) {
			case []interface{}: // TODO type switch doesn't use []map[string]interface{}; just []interface{}
				for _, item := range inputChild {
					if itemAsMap, ok := item.(map[string]interface{}); ok {
						for _, keyConversionListItem := range value {
							err := convertKeys(&itemAsMap, &keyConversionListItem)
							if err != nil {
								return err
							}
						}
					} else {
						return fmt.Errorf("keyConversion doesn't match input structure [1]:\n%v\n%v\n\n%T: %v\n%T: %v", keyConversion, input, value, value, inputChild, inputChild)
					}
				}
			case nil:
				// NOTE: skipping
			default:
				return fmt.Errorf("keyConversion doesn't match input structure [1]:\n%v\n%v\n\n%T: %v\n%T: %v", keyConversion, input, value, value, inputChild, inputChild)
			}
		case map[string]interface{}:
			switch inputChild := (*input)[key].(type) {
			case map[string]interface{}:
				err := convertKeys(&inputChild, &value)
				if err != nil {
					return err
				}
			case nil:
				// NOTE: skipping
			default:
				return fmt.Errorf("keyConversion doesn't match input structure [2]:\n%v\n%v\n\n%T: %v\n%T: %v", keyConversion, input, value, value, inputChild, inputChild)
			}
		default:
			return fmt.Errorf("Unexpected keyConversion format:\n%v", keyConversion)
		}
	}
	return nil
}

func convertKey(input *map[string]interface{}, oldKey string, newKey string) {
	// TODO check newKey doesn't exist yet?
	if _, ok := (*input)[oldKey]; ok {
		(*input)[newKey] = (*input)[oldKey] // TODO will this update work? pointers?
		delete(*input, oldKey)
	}
}

func encloseMaps(input *map[string]interface{}, mapsToEnclose *map[string]interface{}) error {
	// workaround until this is resolved: https://github.com/hashicorp/terraform-plugin-sdk/issues/616
	if mapsToEnclose == nil {
		return nil
	}
	for key, element := range *mapsToEnclose {
		switch value := element.(type) {
		case string: // TODO use other type?
			err := encloseMap(input, key)
			if err != nil {
				return err
			}
		case []map[string]interface{}:
			// TODO check that keyConversion list has a single element? or let it help differentiate list items with differing structure.
			switch inputChild := (*input)[key].(type) {
			case []interface{}: // TODO type switch doesn't use []map[string]interface{}; just []interface{}
				for _, item := range inputChild {
					if itemAsMap, ok := item.(map[string]interface{}); ok {
						for _, mapsToEncloseListItem := range value {
							err := encloseMaps(&itemAsMap, &mapsToEncloseListItem)
							if err != nil {
								return err
							}
						}
					} else {
						return fmt.Errorf("mapsToEnclose doesn't match input structure [1]:\n%v\n%v\n\n%T: %v\n%T: %v", mapsToEnclose, input, value, value, inputChild, inputChild)
					}
				}
			case nil:
				// NOTE: skipping
			default:
				return fmt.Errorf("mapsToEnclose doesn't match input structure [1]:\n%v\n%v\n\n%T: %v\n%T: %v", mapsToEnclose, input, value, value, inputChild, inputChild)
			}
		case map[string]interface{}:
			switch inputChild := (*input)[key].(type) {
			case map[string]interface{}:
				err := encloseMaps(&inputChild, &value)
				if err != nil {
					return err
				}
			case nil:
				// NOTE: skipping
			default:
				return fmt.Errorf("mapsToEnclose doesn't match input structure [2]:\n%v\n%v\n\n%T: %v\n%T: %v", mapsToEnclose, input, value, value, inputChild, inputChild)
			}
		default:
			return fmt.Errorf("Unexpected mapsToEnclose format:\n%v", mapsToEnclose)
		}
	}
	return nil
}

func encloseMap(input *map[string]interface{}, mapKey string) error {
	switch inputChild := (*input)[mapKey].(type) {
	case map[string]interface{}:
		(*input)[mapKey] = []interface{}{(*input)[mapKey]}
		return nil
	case nil:
		return nil
	default:
		return fmt.Errorf("given non-map to enclose, expected map:\n%v\n%v\n\n%T: %v", input, mapKey, inputChild, inputChild)
	}
}

func (c *Client) checkCorrectUser() (bool, error) {
	c.mutexAuth.Lock()
	defer c.mutexAuth.Unlock()
	status, err := c.bwStatus()
	if err != nil {
		return false, err
	}
	if status.ServerUrl != c.Server { // NOTE: server is always known, so always check.
		return false, fmt.Errorf("mismatching serverUrl (%s) and server_url (%s)", status.ServerUrl, c.Server)
	}
	if c.Email != "" && status.UserEmail != c.Email {
		return false, fmt.Errorf("mismatching userEmail (%s) and email (%s)", status.UserEmail, c.Email)
	}
	if c.clientId != "" && status.UserId != strings.TrimPrefix(c.clientId, "user.") {
		return false, fmt.Errorf("mismatching userId (%s) and client_id (%s). client_id should be userId with a 'user.' prefix", status.UserId, c.clientId) // TODO is this always right? presumably no.
	}
	if c.userId != "" && status.UserId != c.userId {
		return false, fmt.Errorf("mismatching userId (%s) and user_id (%s)", status.UserId, c.userId)
	}
	if c.Email == "" {
		c.Email = status.UserEmail
	}
	if c.clientId == "" { // TODO is this always right? presumably no.
		c.clientId = fmt.Sprintf("user.%s", status.UserId)
	}
	if c.userId == "" {
		c.userId = status.UserId
	}
	return true, nil
}

func (c *Client) ensureLoggedInAsCorrectUser() error {
	alreadyLoggedIn, err := c.bwLoginCheck()
	if err != nil {
		return err
	}
	if alreadyLoggedIn {
		correctUser, err := c.checkCorrectUser()
		if err != nil {
			return err
		}
		if correctUser {
			return nil
		} else {
			err = c.ensureLoggedOut()
			if err != nil {
				return err
			}
		}
	}
	err = c.bwLogin()
	if err != nil {
		return err
	}
	correctUser, err := c.checkCorrectUser() // Do this even after logging in; to set the unset params.
	if err != nil || !correctUser {
		return err
	}
	return nil
}

func (c *Client) ensureLoggedOut() error {
	alreadyLoggedIn, err := c.bwLoginCheck()
	if err != nil {
		return err
	}
	if !alreadyLoggedIn {
		return nil
	}
	err = c.bwLogout()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ensureUnlocked() error {
	err := c.ensureLoggedInAsCorrectUser()
	if err != nil {
		return err
	}
	alreadyUnlocked, err := c.bwUnlockCheck()
	if err != nil {
		return err
	}
	if alreadyUnlocked {
		return nil
	}
	err = c.bwUnlock()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ensureLocked() error {
	err := c.ensureLoggedInAsCorrectUser()
	if err != nil {
		return err
	}
	alreadyUnlocked, err := c.bwUnlockCheck()
	if err != nil {
		return err
	}
	if !alreadyUnlocked {
		return nil
	}
	err = c.bwLock()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) runExpectingSuccess(cmd *exec.Cmd, friendlyName string, keyConversion *map[string]interface{}) (*map[string]interface{}, error) {
	// TODO check that --response is in args; if not then add it?
	responseJSONBytes, err := c.runOnly(cmd, friendlyName, 0)
	if err != nil {
		return nil, err
	}
	data, err := c.convertToDataIfSuccessful(responseJSONBytes, friendlyName, keyConversion)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (c *Client) runGivingPasswordExpectingSuccess(cmd *exec.Cmd, friendlyName string, keyConversion *map[string]interface{}) (*map[string]interface{}, error) {
	// TODO check that --response is in args; if not then add it?
	responseJSONBytes, err := c.runAndGivePassword(cmd, friendlyName)
	if err != nil {
		return nil, err
	}
	data, err := c.convertToDataIfSuccessful(responseJSONBytes, friendlyName, keyConversion)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (c *Client) runAndCheckSucceeded(cmd *exec.Cmd, friendlyName string, ignoreCode int) (bool, error) {
	// TODO check that --response is in args; if not then add it?
	responseJSONBytes, err := c.runOnly(cmd, friendlyName, ignoreCode)
	if err != nil {
		return false, err
	}
	response, err := c.convertToResponse(responseJSONBytes, friendlyName, nil)
	if err != nil {
		return false, err
	}
	return (*response).Success, nil // TODO conversion needed?
}

func (c *Client) runOnly(cmd *exec.Cmd, friendlyName string, ignoreCode int) (*[]byte, error) {
	// NOTE: ignoreCode can be 0; so it doesn't ignore any errors.
	output, err := cmd.CombinedOutput()
	if err != nil { // TODO combine these if statements?
		if exitError, ok := err.(*exec.ExitError); !ok || exitError.ExitCode() != ignoreCode {
			return &output, fmt.Errorf("cannot %s: %s\nexit code: %s", friendlyName, string(output), err)
		}
	}
	return &output, nil
}

func (c *Client) runAndGivePassword(cmd *exec.Cmd, friendlyName string) (*[]byte, error) {
	// NOTE: password given as cli args or env vars is generally less secure than stdin. Simplify if that isn't true in this case.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	go func() {
		defer stdin.Close() // TODO handle
		if _, err := io.WriteString(stdin, fmt.Sprintf("%s\n", c.MasterPassword)); err != nil {
			log.Println("[ERROR] ", err)
		}
	}()

	output, err := c.runOnly(cmd, friendlyName, 0)
	if err != nil {
		return nil, err
	}
	outputString := string(*output)
	jsonStartIndex := strings.Index(outputString, "{") // TODO will this break if the password has a { in it?
	if jsonStartIndex > 0 {
		outputString = outputString[jsonStartIndex:]
	}
	if !strings.HasPrefix(outputString, "{") {
		return nil, fmt.Errorf("failed to trim interactive prompts from output:\n%s\n-->\n%s", string(*output), outputString)
	}
	trimmedOutput := []byte(outputString)
	return &trimmedOutput, nil
}

func (c *Client) convertToResponse(responseJSONBytes *[]byte, friendlyName string, keyConversion *map[string]interface{}) (*Response, error) {
	// NOTE: I think that when success is false, message is populated, otherwise data. any exceptions? can check with this.
	var response Response
	data, err := UnmarshalConvertKeys(responseJSONBytes, keyConversion)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal response from %s: %s\nexit code: %s", friendlyName, string(*responseJSONBytes), err)
	}
	err = mapstructure.Decode(data, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) convertToDataIfSuccessful(responseJSONBytes *[]byte, friendlyName string, keyConversion *map[string]interface{}) (*map[string]interface{}, error) {
	response, err := c.convertToResponse(responseJSONBytes, friendlyName, keyConversion)
	if err != nil {
		return nil, err
	}
	if response.Success {
		return &response.Data, nil
	} else {
		return nil, fmt.Errorf("unsuccessful %s: %s", friendlyName, response.Message)
	}
}
