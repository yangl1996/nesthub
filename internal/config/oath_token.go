package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/yangl1996/nesthub/internal/helpers"
	"golang.org/x/oauth2"
)

func (cfg *Config) NewOAuthTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	token := &oauth2.Token{}
	if err := helpers.JsonUnmarshalFile(cfg.OAuthTokenPath, token); err != nil {
		return nil, fmt.Errorf("failed to load oauth token: %w", err)
	}

	tokenCfg := cfg.getOAuthConfig()

	return tokenCfg.TokenSource(ctx, token), nil
}

func (cfg *Config) WriteOAuthTokenToFile(authCode, path string) error {
	oauthConfig := cfg.getOAuthConfig()
	ctx := context.Background()

	token, err := oauthConfig.Exchange(ctx, authCode)
	if err != nil {
		return fmt.Errorf("failed to convert authorization code into a token: %w", err)
	}

	tokenJson, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(cfg.OAuthTokenPath, tokenJson, 0o600); err != nil {
		return fmt.Errorf("failed to write token to file %s: %w", cfg.OAuthTokenPath, err)
	}

	return nil
}

func (cfg *Config) getOAuthConfig() oauth2.Config {
	return oauth2.Config{
		ClientID:     cfg.OAuthClientID,
		ClientSecret: cfg.OAuthClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: "https://oauth2.googleapis.com/token",
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
		},
		RedirectURL: cfg.SetupRedirectUri,
	}
}
