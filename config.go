package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
	SDMProjectID string		// the project ID shown in the SDM console
	OAuthClientID string	// the oauth client ID created in GCP and set in SDM project
	OAuthClientSecret string	// the oauth client secret created in GCP
	AccessToken string	// the oauth access token obtained in the initial token request
	RefreshToken string	// the oauth refresh token obtained in the initial token request
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
	err = json.Unmarshal(b, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}
