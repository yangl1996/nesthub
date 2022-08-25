package main

import (
	"flag"
	"log"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	"github.com/yangl1996/nesthub/internal/config"
	"github.com/yangl1996/nesthub/internal/onboard"
	"github.com/yangl1996/nesthub/pkg/emulation"
)

func main() {
	doSetupFlag := flag.Bool("setup", false, "go through the setup routine")
	configPathFlag := flag.String("config", "config.json", "path to the config file")
	flag.Parse()

	configPath := "config.json"
	if configPathFlag != nil {
		configPath = *configPathFlag
	}

	c, err := config.Parse(configPath)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	if *doSetupFlag {
		if err := onboard.Setup(c); err != nil {
			log.Fatalf("failed to setup: %v", err)
		}
	}

	svc := service.NewThermostat()
	if _, err := emulation.NewEmulatedDevice(svc, c); err != nil {
		log.Fatalln(err)
	}
	log.Println("Device emulation started")

	// init the bridge device
	info := accessory.Info{
		Name:         c.HubName,
		Manufacturer: "leiy",
	}
	acc := accessory.NewBridge(info)

	// add the service to the bridge
	acc.AddService(svc.Service)
	t, err := hc.NewIPTransport(
		hc.Config{
			Pin:         c.PairingCode,
			StoragePath: c.StoragePath,
			Port:        c.Port,
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
