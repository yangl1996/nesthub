package config

import (
	"errors"
	"fmt"

	"github.com/yangl1996/nesthub/internal/helpers"
)

type Config struct {
	// HubName is the name of the hub
	HubName string `json:"HubName,omitempty"`

	// SDMProjectID is the project ID shown in the SDM console
	SDMProjectID string `json:"SDMProjectID,omitempty"`

	// GCPProjectID is the project ID shown in the GCP console
	GCPProjectID string `json:"GCPProjectID,omitempty"`

	// OAuthClientID is the oauth client ID created in GCP and set in SDM project
	OAuthClientID string `json:"OAuthClientID,omitempty"`

	// OAuthClientSecret is the oauth client secret created in GCP
	OAuthClientSecret string `json:"OAuthClientSecret,omitempty"`

	// OAuthTokenPath is the path to the oauth token
	OAuthTokenPath string `json:"OAuthToken,omitempty"`

	// ServiceAccountKey credentials of the service account of GCP project
	ServiceAccountKey string `json:"ServiceAccountKey,omitempty"`

	// PairingCode is the 8 digit pairing code
	PairingCode string `json:"PairingCode,omitempty"`

	// Port is the port that homekit will connect to
	Port string `json:"Port,omitempty"`

	// StoragePath is the filepath where connection data is stored
	StoragePath string `json:"StoragePath,omitempty"`

	// An http server is required during nesthub setup, you can use this field to specify a
	// network address to use. (default: http://localhost:7979)
	SetupRedirectUri string `json:"SetupRedirectUri,omitempty"`
}

func NewConfig(path string) (*Config, error) {
	cfg := &Config{}
	if err := helpers.JsonUnmarshalFile(path, cfg); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := cfg.validateRequiredFields(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cfg.populateOptionalFields()

	return cfg, nil
}

// validateRequiredFields checks that all required config fields exist
func (cfg *Config) validateRequiredFields() error {
	errs := []error{}

	if cfg.HubName == "" {
		errs = append(errs, errors.New("HubName"))
	}

	if cfg.SDMProjectID == "" {
		errs = append(errs, errors.New("SDMProjectID"))
	}

	if cfg.GCPProjectID == "" {
		errs = append(errs, errors.New("GCPProjectID"))
	}

	if cfg.OAuthClientID == "" {
		errs = append(errs, errors.New("OAuthClientID"))
	}

	if cfg.OAuthClientSecret == "" {
		errs = append(errs, errors.New("OAuthClientSecret"))
	}

	if cfg.OAuthTokenPath == "" {
		errs = append(errs, errors.New("OAuthToken"))
	}

	if cfg.ServiceAccountKey == "" {
		errs = append(errs, errors.New("ServiceAccountKey"))
	}

	return helpers.ErrListToErr("config missing fields", errs)
}

func (cfg *Config) populateOptionalFields() {
	if cfg.SetupRedirectUri == "" {
		cfg.SetupRedirectUri = "http://localhost:7979"
	}
}
