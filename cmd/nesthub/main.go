package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	"github.com/yangl1996/nesthub/internal/config"
	"github.com/yangl1996/nesthub/internal/helpers"
	"github.com/yangl1996/nesthub/internal/onboard"
	"github.com/yangl1996/nesthub/pkg/emulation"
)

const sdmSvcName = "smartdevicemanagement.googleapis.com"

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

		if err := os.RemoveAll(cfg.StoragePath); err != nil {
			log.Fatal(err)
		}
	}

	svc := service.NewThermostat()
	if _, err := emulation.NewEmulatedDevice(ctx, cfg, svc); err != nil {
		log.Fatalf("failed to create emulated device: %s", err)
	}

	log.Println("Device emulation started")

	// init the bridge device
	info := accessory.Info{
		Name:         cfg.HubName,
		Manufacturer: "leiy",
	}
	acc := accessory.NewBridge(info)

	// add the service to the bridge
	acc.AddService(svc.Service)

	t, err := hc.NewIPTransport(
		hc.Config{
			Pin:         cfg.PairingCode,
			StoragePath: cfg.StoragePath,
			Port:        cfg.Port,
		},
		acc.Accessory,
	)
	if err != nil {
		log.Fatalf("failed to start transport: %s", err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
