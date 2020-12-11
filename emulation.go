package main

import (
	"cloud.google.com/go/pubsub"
	"golang.org/x/oauth2"
	sdm "google.golang.org/api/smartdevicemanagement/v1"
	"google.golang.org/api/option"
	"context"
	"log"
	"time"
)

type EmulatedDevice struct {
	*DeviceEndpoint
	sub *pubsub.Subscription
}

func NewEmulatedDevice(c Config) (*EmulatedDevice, error) {
	e := &EmulatedDevice{}
	ctx := context.Background()

	// get the oauth2 token
	config := oauth2.Config {
		ClientID: c.OAuthClientID,
		ClientSecret: c.OAuthClientSecret,
		Endpoint: oauth2.Endpoint {
			TokenURL: "https://oauth2.googleapis.com/token",
			AuthURL: "https://accounts.google.com/o/oauth2/auth",
		},
		RedirectURL: "https://www.google.com",
	}
	token := oauth2.Token {
		AccessToken: c.AccessToken,
		TokenType: "Bearer",
		RefreshToken: c.RefreshToken,
		Expiry: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	source := config.TokenSource(ctx, &token)
	s, err := sdm.NewService(ctx, option.WithTokenSource(source))
	if err != nil {
		return e, err
	}

	// list the devices
	resp, err := s.Enterprises.Devices.List("enterprises/"+c.SDMProjectID).Do()
	if err != nil {
		return e, err
	}
	log.Println("Retrieved", len(resp.Devices), "devices")
	if len(resp.Devices) > 1 {
		log.Fatalln("Do not support multiple devices for now")
	}
	// FIXME: I'm being lazy here by only supporting one device and not checking
	// the type of the device. Works for me now.
	dn := resp.Devices[0].Name
	log.Println("Controlling device", dn)
	de := &DeviceEndpoint {
		Service: s,
		Name: dn,
	}
	e.DeviceEndpoint = de

	// create pubsub client and subscription
	pc, err := pubsub.NewClient(ctx, c.GCPProjectID, option.WithCredentialsFile(c.ServiceAccountKey))
	if err != nil {
		return e, err
	}
	sub := pc.Subscription(c.PubSubID)
	e.sub = sub
	return e, nil
}

func (d *EmulatedDevice) ListenEvents() error {
	// create a pubsub client
	ctx := context.Background()
	for ;; {
		_ = d.sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			log.Printf("Got message: %s", m.Data)
			m.Ack()
		})
	}
	return nil
}

