package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	"log"
	"flag"
)

const (
	Off = iota
	Heat
	Cool
	Auto
)

func main() {
	doSetupFlag := flag.Bool("setup", false, "go through the setup routine")
	flag.Parse()

	c, err := parse("config.json")
	if err != nil {
		log.Fatalln(err)
	}

	if *doSetupFlag {
		err = setup(c)
		if err != nil {
			log.Fatalln(err)
		}
	}

	svc := service.NewThermostat()
	_, err = NewEmulatedDevice(svc, c)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Device emulation started")

	// try to get the current temperature
	// init the bridge device
	info := accessory.Info{
		Name:         c.HubName,
		Manufacturer: "leiy",
	}
	acc := accessory.NewBridge(info)

	// add the service to the bridge
	acc.AddService(svc.Service)

	t, err := hc.NewIPTransport(hc.Config{Pin: c.PairingCode}, acc.Accessory)
	if err != nil {
		log.Fatalln(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
