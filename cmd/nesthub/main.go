package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/service"
	"github.com/yangl1996/nesthub/internal/config"
	"github.com/yangl1996/nesthub/internal/helpers"
	"github.com/yangl1996/nesthub/internal/onboard"
	"github.com/yangl1996/nesthub/pkg/emulation"
)

func main() {
	ctx := context.Background()

	configPathFlag := flag.String("config", "config.json", "path to the config file")
	flag.Parse()

	// TODO if config doesn't exist, template the config
	// Confirm config is valid
	cfg, err := config.NewConfig(*configPathFlag)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Confirm SDM is enabled
	const sdmSvcName = "smartdevicemanagement.googleapis.com"
	if err := onboard.SvcEnabled(ctx, cfg, sdmSvcName); errors.Is(err, helpers.ErrSvcNotEnabled) {
		log.Println("smart device management service not enabled")

		if err := onboard.EnableSvc(ctx, cfg, sdmSvcName); err != nil {
			log.Fatalf("failed to enable smart device management service: %v", err)
		}
	} else if err != nil {
		log.Fatalf("failed to check if smart device management service is enabled: %v", err)
	}

	// Confirm oauth token is valid
	if _, err := cfg.NewOAuthTokenSource(ctx); err != nil {
		log.Println("invalid or missing oauth token")

		if err := onboard.AuthorizeOAuthToken(ctx, cfg); err != nil {
			log.Fatalf("failed to authorize oath token: %v", err)
		}
	}

	s := service.NewThermostat()
	a := accessory.NewBridge(accessory.Info{
		Name:         cfg.HubName,
		Manufacturer: "github.com/yangl1996/nesthub",
	})

	a.AddS(s.S)

	if _, err := emulation.NewEmulatedDevice(ctx, cfg, s); err != nil {
		log.Fatalf("failed to create emulated device: %s", err)
	}

	log.Println("Device emulation started")

	fs := hap.NewFsStore(cfg.StoragePath)

	server, err := hap.NewServer(fs, a.A)
	if err != nil {
		log.Fatalf("failed to start transport: %s", err)
	}

	server.Pin = cfg.PairingCode
	// server.Addr = cfg.Address

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-c
		signal.Stop(c)
		cancel()
	}()

	// Run the server
	fmt.Printf("Server exited: %s\n", server.ListenAndServe(ctx))
}
