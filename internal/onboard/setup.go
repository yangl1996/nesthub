package onboard

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/yangl1996/nesthub/internal/config"
	"github.com/yangl1996/nesthub/internal/helpers"
	"google.golang.org/api/option"
	su "google.golang.org/api/serviceusage/v1"
)

func SvcEnabled(ctx context.Context, cfg *config.Config, svcName string) error {
	s, err := su.NewService(ctx, option.WithCredentialsFile(cfg.ServiceAccountKey))
	if err != nil {
		return fmt.Errorf("failed to create Service Usage client: %w", err)
	}

	parent := fmt.Sprintf("projects/%s", cfg.GCPProjectID)
	svc := fmt.Sprintf("%s/services/%s", parent, svcName)

	resp, err := s.Services.BatchGet(parent).Names(svc).Do()
	if err != nil {
		return fmt.Errorf("failed to get the status of service %s: %w", svcName, err)
	}

	for _, svc := range resp.Services {
		if svc.Name == svcName && svc.State != "ENABLED" {
			return helpers.ErrSvcNotEnabled
		}
	}

	return nil
}

func EnableSvc(ctx context.Context, cfg *config.Config, svcName string) error {
	log.Printf("Enabling service %s", svcName)

	s, err := su.NewService(ctx, option.WithCredentialsFile(cfg.ServiceAccountKey))
	if err != nil {
		return fmt.Errorf("failed to create Service Usage client: %w", err)
	}

	req := &su.BatchEnableServicesRequest{
		ServiceIds: []string{svcName},
	}

	op, err := s.Services.BatchEnable("projects/"+cfg.GCPProjectID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to enable the service %s: %w", svcName, err)
	}

	for {
		if op.Done {
			if op.Error == nil {
				break
			} else {
				return fmt.Errorf("failed to enable the service %s: %v", svcName, op.Error)
			}
		} else {
			time.Sleep(1 * time.Second)
			log.Println("Waiting for the service to be enabled...")
			if op, err = s.Operations.Get(op.Name).Do(); err != nil {
				return fmt.Errorf("failed to get status of service enablement %s: %w", op.Name, err)
			}
		}
	}

	log.Println("Service enabled")

	return nil
}

func AuthorizeOAuthToken(ctx context.Context, cfg *config.Config) error {
	log.Println("Authorizing oauth token")

	authCode := ""
	authURL := fmt.Sprintf(
		"https://nestservices.google.com/partnerconnections/%s/auth?redirect_uri=%s&access_type=offline&prompt=consent&client_id=%s&response_type=code&scope=https://www.googleapis.com/auth/sdm.service",
		cfg.SDMProjectID,
		cfg.SetupRedirectUri,
		cfg.OAuthClientID,
	)

	authDone := &sync.WaitGroup{}

	u, err := url.Parse(cfg.SetupRedirectUri)
	if err != nil {
		return fmt.Errorf("error parsing redirect uri: %w", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		keys := r.URL.Query()

		authCode = keys.Get("code")
		if len(authCode) == 0 {
			fmt.Fprintf(w, "Bad redirect.")
		}

		fmt.Fprintf(w, "Successful authorization. Please go back to Terminal.")

		authDone.Done()
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", u.Port()),
		ReadHeaderTimeout: 1 * time.Second,
	}

	http.HandleFunc("/", handler)

	// start the server to receive callback from the browser
	go srv.ListenAndServe() //nolint:errcheck

	// send user to the browser to authorize the token
	if err := helpers.OpenURL(authURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	// wait for authorization to finish
	authDone.Add(1)
	authDone.Wait()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown the server: %w", err)
	}

	log.Println("Authorization successful")

	if err := cfg.WriteOAuthTokenToFile(authCode, cfg.OAuthTokenPath); err != nil {
		return err
	}

	return nil
}
