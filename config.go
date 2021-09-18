package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"golang.org/x/oauth2"
)

type Config struct {
	SDMProjectID      string // the project ID shown in the SDM console
	OAuthClientID     string // the oauth client ID created in GCP and set in SDM project
	OAuthClientSecret string // the oauth client secret created in GCP
	GCPProjectID      string // the project ID shown in the GCP console
	ServiceAccountKey string // credentials of the service account of GCP project
	OAuthToken        string // path to the oauth token
	HubName           string // name of the hub
	PairingCode       string // 8 digits of pairing code
	StoragePath       string // nesthub will store data at this path
}

func parse(path string) (Config, error) {
	var c Config
	jsonFile, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer jsonFile.Close()
	b, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

func (c Config) oauthConfig() oauth2.Config {
	// get the oauth2 token
	config := oauth2.Config{
		ClientID:     c.OAuthClientID,
		ClientSecret: c.OAuthClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: "https://oauth2.googleapis.com/token",
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
		},
		RedirectURL: "http://localhost:7979",
	}
	return config
}

func (c Config) oauthToken() (oauth2.Token, error) {
	t := oauth2.Token{}
	jsonFile, err := os.Open(c.OAuthToken)
	if err != nil {
		return t, err
	}
	defer jsonFile.Close()
	b, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return t, err
	}
	return t, nil
}
