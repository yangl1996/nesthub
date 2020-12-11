package main

import (
	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
	"context"
	"log"
)

func ListenEvents(pid string, subid string, credfile string) error {
	// create a pubsub client
	ctx := context.Background()
	c, err := pubsub.NewClient(ctx, pid, option.WithCredentialsFile(credfile))
	if err != nil {
		return err
	}
	sub := c.Subscription(subid)
	for ;; {
		_ = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			log.Printf("Got message: %s", m.Data)
			m.Ack()
		})
	}
	return nil
}
