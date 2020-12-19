package main

import (
	"google.golang.org/api/option"
	su "google.golang.org/api/serviceusage/v1"
	"log"
	"context"
	"errors"
	"time"
)

func setup(projectName, path string) error {
	ctx := context.Background()
	log.Println("Enabling Smart Device Management API")
	// init config for service usage API
	s, err := su.NewService(ctx, option.WithCredentialsFile(path))
	if err != nil {
		return err
	}
	// create the request
	req := &su.BatchEnableServicesRequest {
		ServiceIds: []string{"smartdevicemanagement.googleapis.com"},
	}
	op, err := s.Services.BatchEnable("projects/"+projectName, req).Do()
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

	return nil
}
