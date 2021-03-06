package main

import (
	"google.golang.org/api/option"
	//"cloud.google.com/go/pubsub"
	"encoding/json"
	su "google.golang.org/api/serviceusage/v1"
	"log"
	"context"
	"errors"
	"time"
	"io/ioutil"
	"net/http"
	"fmt"
	"sync"
)

func setup(config Config) error {
	ctx := context.Background()
	log.Println("Enabling Smart Device Management API")
	// init config for service usage API
	s, err := su.NewService(ctx, option.WithCredentialsFile(config.ServiceAccountKey))
	if err != nil {
		return err
	}
	// create the request
	req := &su.BatchEnableServicesRequest {
		ServiceIds: []string{"smartdevicemanagement.googleapis.com"},
	}
	op, err := s.Services.BatchEnable("projects/"+config.GCPProjectID, req).Do()
	if err != nil {
		return err
	}
	opName := op.Name
	// poll the operation to wait for the result
	for ;; {
		if op.Done == true {
			if op.Error == nil {
				break
			} else {
				return errors.New("failed enabling SDM API")
			}
		} else {
			time.Sleep(1 * time.Second)
			op, err = s.Operations.Get(opName).Do()
			if err != nil {
				return err
			}
		}
	}

	// assume that the user has already created an oauth 2.0 client ID
	authURL := "https://nestservices.google.com/partnerconnections/"+config.SDMProjectID+"/auth?redirect_uri=http://localhost:7979&access_type=offline&prompt=consent&client_id="+config.OAuthClientID+"&response_type=code&scope=https://www.googleapis.com/auth/sdm.service"
	authCode := ""
	authDone := &sync.WaitGroup{}
	authDone.Add(1)

	// start the server to receive callback from the browser
	handler := func(w http.ResponseWriter, r *http.Request) {
		keys := r.URL.Query()
		authCode = keys.Get("code")
		if len(authCode) == 0 {
			fmt.Fprintf(w, "Bad redirect.")
		} else {
			fmt.Fprintf(w, "Successful authorization. Please go back to Terminal.")
			defer authDone.Done()
		}
	}
	srv := &http.Server{Addr: ":7979"}
    http.HandleFunc("/", handler)
    go srv.ListenAndServe()
	// let the user login
	err = openURL(authURL)
	// wait for authorization to finish
	authDone.Wait()
	if err := srv.Shutdown(context.Background()); err != nil {
		return err
	}
	log.Println("Authorization successful.", authCode)
	// exchange to obtain the token
	oauthConfig := config.oauthConfig()
	token, err := oauthConfig.Exchange(ctx, authCode)
	if err != nil {
		return err
	}
	tokenJson, err := json.Marshal(token)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(config.OAuthToken, tokenJson, 0644)
	if err != nil {
		return err
	}

	/*
	// create a pubsub
	pc, err := pubsub.NewClient(ctx, config.GCPProjectID, option.WithCredentialsFile(config.ServiceAccountKey))
	if err != nil {
		return err
	}
	sub := pc.Subscription("homebridge-pubsub")
	subThere, err := sub.Exists(ctx)
	if err != nil {
		return err
	}
	if subThere == true {
		err = sub.Delete(ctx)
		if err != nil {
			return err
		}
	}
	_, err = pc.CreateSubscription(ctx, "homebridge-pubsub", pubsub.SubscriptionConfig{
		// nest pubsub topics are of format projects/sdm-prod/topics/enterprise-<SDM project ID>
		Topic:            pc.TopicInProject("enterprise-"+config.SDMProjectID, "sdm-prod"),
		AckDeadline:      10 * time.Second,
		ExpirationPolicy: 1 * time.Hour,
	})
	if err != nil {
		return err
	}
	log.Println("Pubsub subscription created")
	*/

	return nil
}

