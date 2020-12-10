package main

import (
	"golang.org/x/oauth2"
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	sdm "google.golang.org/api/smartdevicemanagement/v1"
	"google.golang.org/api/option"
	"context"
	"log"
	"os"
	"time"
)

const (
	Off = iota
	Heat
	Cool
	Auto
)

func main() {
	projID := os.Getenv("NEST_PROJECT_ID")
	ctx := context.Background()

	// get the oauth2 token
	config := oauth2.Config {
		ClientID: "9779670944-3lvn9q08gqih9b3scbrcpvef42lgvkg8.apps.googleusercontent.com",
		ClientSecret: "iXWrA0vjywPL9Xpyqw2Rxd-C",
		Endpoint: oauth2.Endpoint {
			TokenURL: "https://oauth2.googleapis.com/token",
			AuthURL: "https://accounts.google.com/o/oauth2/auth",
		},
		RedirectURL: "https://www.google.com",
	}
	token := oauth2.Token {
		AccessToken: "ya29.a0AfH6SMA9LKIehYhq0rop6JgjMTsGClRPt5ln0KNbi3SBvl_GaO1q6VjQFKCL6WPcHBIsi1RKRxWe7lGGLJlM4qQj5Da-8QmyvtfHwX4MO6Ziu-fupPgnHziZ8tif0Q9mPsYsAGSupLUtg_MfJTGGQa6xjiCL2XGLcZT09ayubEs",
		TokenType: "Bearer",
		RefreshToken: "1//0dXCgw0Zg0ZShCgYIARAAGA0SNwF-L9IrZDP8IaUiZWtedJfRqn59szI6r_rdnmniOxMEI7EpPvpyBjRV2uEF5xK5IA0PIUHKzVg",
		Expiry: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	source := config.TokenSource(ctx, &token)
	s, err := sdm.NewService(ctx, option.WithTokenSource(source))
	if err != nil {
		log.Fatal(err)
	}
	resp, err := s.Enterprises.Devices.List("enterprises/"+projID).Do()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("retrieved", len(resp.Devices), "devices")

	temp := 20.0
	cel := true
	targets := Auto
	currents := Heat

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