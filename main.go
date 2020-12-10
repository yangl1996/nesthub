package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	"log"
)

func turnLightOn() {
	log.Println("Turn Light On")
}

func turnLightOff() {
	log.Println("Turn Light Off")
}

const (
	Off = iota
	Heat
	Cool
	Auto
)


func main() {
	temp := 20.0
	cel := true
	targets := Auto
	currents := Heat

	// init the bridge device
	info := accessory.Info{
		Name:         "NestHub",
		Manufacturer: "Lei Yang",
	}
	acc := accessory.NewBridge(info)
	// init the thermostat service
	svc := service.NewThermostat()

	// set the characteristics
	// Celsius is 0, Fahrenheit is 1
	// https://developer.apple.com/documentation/homekit/hmcharacteristicvaluetemperatureunit
	// Off is 0, Heat is 1, Cool is 2, Auto is 3
	// https://developer.apple.com/documentation/homekit/hmcharacteristicvalueheatingcooling
	// Note that TargetHeatingCoolingState can be 0-3, but CurrentHeatingCoolingState
	// can only be 0-2, because "Auto" is not an actual state.
	// https://github.com/homebridge/HAP-NodeJS/issues/815
	// Reported values must always be in Celsius
	// Another good reference of all those stuff is
	// https://github.com/brutella/hc/blob/master/gen/metadata.json
	svc.TargetTemperature.OnValueRemoteGet(func() float64 {
		return temp
	})
	svc.TargetTemperature.OnValueRemoteUpdate(func(n float64) {
		temp = n
		log.Println("temp set to", n)
		return
	})

	svc.CurrentTemperature.OnValueRemoteGet(func() float64 {
		return temp + 1.5
	})

	svc.TemperatureDisplayUnits.OnValueRemoteGet(func() int {
		if cel == true {
			return 0
		} else {
			return 1
		}
	})
	svc.TemperatureDisplayUnits.OnValueRemoteUpdate(func(n int) {
		if n == 0 {
			cel = true
		} else {
			cel = false
		}
		log.Println("unit set to", n)
		return
	})

	svc.TargetHeatingCoolingState.OnValueRemoteGet(func() int {
		return targets
	})
	svc.TargetHeatingCoolingState.OnValueRemoteUpdate(func(n int) {
		targets = n
		log.Println("target mode set to", n)
		return
	})

	svc.CurrentHeatingCoolingState.OnValueRemoteGet(func() int {
		return currents
	})

	// add the service to the bridge
	acc.AddService(svc.Service)

	t, err := hc.NewIPTransport(hc.Config{Pin: "77887788"}, acc.Accessory)
	if err != nil {
		log.Fatal(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
