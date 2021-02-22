package bitwarden

import (
	"fmt"
	"sync"

	"os/exec"
)

type Client struct { // TODO need to isolate this from environment vars so it doesn't use host login context.
	// TODO is 2fa required again for unlock or sync?
	// TODO default cmd has --nointeraction
	userId             string
	clientId           string
	clientSecret       string
	Email              string // TODO needed? if I separate the env, will I ever need to login after the first time?
	MasterPassword     string
	SessionKey         string
	Server             string
	BitwardenCLIBinary string
	mutex              *sync.Mutex
	mutexAuth          *sync.Mutex // Blocks changes to login/logout unlock/lock. See if I can adjust this so multiple simultaneous operations can run while each separately blocking auth changes
}

func NewClient(email string, masterPassword string, server string, clientId string, clientSecret string, userId string, sessionKey string) (*Client, error) {
	bin, err := findHostBitwardenCLI()
	if err != nil {
		return nil, fmt.Errorf("bw (Bitwarden CLI) not found")
	}

	bw := &Client{
		userId:             userId,
		clientId:           clientId,
		clientSecret:       clientSecret,
		Email:              email,
		MasterPassword:     masterPassword,
		SessionKey:         sessionKey,
		Server:             server,
		BitwardenCLIBinary: bin,
		mutex:              &sync.Mutex{},
		mutexAuth:          &sync.Mutex{},
	}
	err = bw.ensureUnlocked()
	if err != nil {
		return bw, err
	}
	err = bw.bwSync()
	if err != nil {
		return bw, err
	}
	return bw, nil
}

func findHostBitwardenCLI() (string, error) {
	// TODO constrain version
	o, err := exec.Command("bw", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("Error from bw:\n%s", o)
	}
	return "bw", nil
}
