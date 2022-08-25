package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/yangl1996/nesthub/internal/config"
	"github.com/yangl1996/nesthub/internal/helpers"
	"google.golang.org/api/option"
	su "google.golang.org/api/serviceusage/v1"
)

func Setup(config config.Config) error {
	ctx := context.Background()
	log.Println("Enabling Smart Device Management API")
	// init config for service usage API
	s, err := su.NewService(ctx, option.WithCredentialsFile(config.ServiceAccountKey))
	if err != nil {
		return fmt.Errorf("failed to create Google Service Usage API client: %v", err)
	}

	// create the request
	req := &su.BatchEnableServicesRequest{
		ServiceIds: []string{"smartdevicemanagement.googleapis.com"},
	}
	op, err := s.Services.BatchEnable("projects/"+config.GCPProjectID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to enable the following Google services %v: %v", req.ServiceIds, err)
	}

	// poll the operation to wait for the result
	for {
		if op.Done {
			if op.Error == nil {
				break
			} else {
				return fmt.Errorf("failed to enable the Smart Device Management API: %v", op.Error)
			}
		} else {
			time.Sleep(1 * time.Second)
			log.Println("Waiting for the Smart Device Management API to be enabled...")
			if op, err = s.Operations.Get(op.Name).Do(); err != nil {
				return fmt.Errorf("failed to get state of Smart Device Management API Enablement operation %v: %v", op.Name, err)
			}
		}
	}

	// assume that the user has already created an oauth 2.0 client ID
	authURL := "https://nestservices.google.com/partnerconnections/" + config.SDMProjectID + "/auth?redirect_uri=http://localhost:7979&access_type=offline&prompt=consent&client_id=" + config.OAuthClientID + "&response_type=code&scope=https://www.googleapis.com/auth/sdm.service"
	authCode := ""
	authDone := &sync.WaitGroup{}
	authDone.Add(1)
	defer authDone.Done()

	// start the server to receive callback from the browser
	handler := func(w http.ResponseWriter, r *http.Request) {
		keys := r.URL.Query()
		authCode = keys.Get("code")
		if len(authCode) == 0 {
			fmt.Fprintf(w, "Bad redirect.")
		}
		fmt.Fprintf(w, "Successful authorization. Please go back to Terminal.")
	}
	srv := &http.Server{
		Addr:              ":7979",
		ReadHeaderTimeout: 1 * time.Second,
	}
	http.HandleFunc("/", handler)
	go srv.ListenAndServe() //nolint:errcheck

	// let the user login
	if err := helpers.OpenURL(authURL); err != nil {
		return fmt.Errorf("failed to open browser: %v", err)
	}

	// wait for authorization to finish
	authDone.Wait()
	if err := srv.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("failed to shutdown the server: %v", err)
	}
	log.Println("Authorization successful.", authCode)

	// exchange to obtain the token
	oauthConfig := config.OauthConfig()
	token, err := oauthConfig.Exchange(ctx, authCode)
	if err != nil {
		return fmt.Errorf("failed to convert authorization code into a token: %v", err)
	}
	tokenJson, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %v", err)
	}

	if err := os.WriteFile(config.OAuthToken, tokenJson, 0600); err != nil {
		return fmt.Errorf("failed to write token to file %s: %v", config.OAuthToken, err)
	}

	return nil
}
