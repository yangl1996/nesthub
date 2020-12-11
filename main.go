package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	"log"
)

const (
	Off = iota
	Heat
	Cool
	Auto
)

func main() {
	c, err := parse("config.json")

	e, err := NewEmulatedDevice(c)
	if err != nil {
		log.Fatalln(err)
	}
	e.ListenEvents()

	// try to get the current temperature
	res, err := e.GetDevice()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Current device state:", res)

	// init the bridge device
	info := accessory.Info{
		Name:         "Nest Hub",
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
	
	// helper function to report error
	getDevice := func() DeviceTraits {
		r, err := e.GetDevice()
		if err != nil {
			log.Println(err)
		}
		return r
	}
	svc.TargetTemperature.OnValueRemoteGet(func() float64 {
		// depends on the set mode
		r := getDevice()
		switch r.SetMode.Mode {
			case "OFF": return 0
			case "HEAT": return r.SetTemp.HeatCelsius
			case "COOL": return r.SetTemp.CoolCelsius
			case "HEATCOOL": return (r.SetTemp.HeatCelsius + r.SetTemp.CoolCelsius) / 2.0
		}
		panic("unreachable target temp")
		return 0
	})
	svc.TargetTemperature.OnValueRemoteUpdate(func(n float64) {
		r := getDevice()
		var err error
		switch r.SetMode.Mode {
			case "OFF": err = nil
			case "HEAT": err = e.SetHeat(n)
			case "COOL": err = e.SetCool(n)
			case "HEATCOOL": err = e.SetHeatCool(n-2.5, n+2.5)
		}
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("temp set to", n)
		return
	})

	svc.CurrentTemperature.OnValueRemoteGet(func() float64 {
		return getDevice().CurrTemp.TempCelsius
	})

	svc.TemperatureDisplayUnits.OnValueRemoteGet(func() int {
		unit := getDevice().DisplayUnit.Unit
		switch unit {
		case "CELSIUS": return 0
		case "FAHRENHEIT": return 1
		}
		panic("unreachable unit")
		return 0
	})
	/*
	// SDM does not support changing the display unit
	svc.TemperatureDisplayUnits.OnValueRemoteUpdate(func(n int) {
		if n == 0 {
			cel = true
		} else {
			cel = false
		}
		log.Println("unit set to", n)
		return
	})
	*/

	svc.TargetHeatingCoolingState.OnValueRemoteGet(func() int {
		r := getDevice()
		switch r.SetMode.Mode {
			case "OFF": return 0
			case "HEAT": return 1
			case "COOL": return 2
			case "HEATCOOL": return 3
		}
		panic("unreachable target mode")
		return 0
	})
	svc.TargetHeatingCoolingState.OnValueRemoteUpdate(func(n int) {
		var s string
		switch n {
			case 0: s = "OFF"
			case 1: s = "HEAT"
			case 2: s = "COOL"
			case 3: s = "HEATCOOL"
		}
		err := e.SetMode(s)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("target mode set to", n)
		return
	})

	svc.CurrentHeatingCoolingState.OnValueRemoteGet(func() int {
		mode := getDevice().CurrMode.Status
		switch mode {
		case "OFF": return 0
		case "HEATING": return 1
		case "COOLING": return 2
		}
		panic("unreachable current mode")
		return 0
	})

	// add the service to the bridge
	acc.AddService(svc.Service)

	t, err := hc.NewIPTransport(hc.Config{Pin: "77887788"}, acc.Accessory)
	if err != nil {
		log.Fatalln(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
