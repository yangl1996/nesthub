package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/brutella/hc/service"
	"google.golang.org/api/option"
	sdm "google.golang.org/api/smartdevicemanagement/v1"
)

const (
	OFF      = "OFF"
	HEAT     = "HEAT"
	COOL     = "COOL"
	HEATCOOL = "HEATCOOL"
)

type PubsubUpdate struct {
	Timestamp      time.Time
	ResourceUpdate struct {
		Traits DeviceTraits
	}
}

type EmulatedDevice struct {
	*DeviceEndpoint
	sub *pubsub.Subscription
	*sync.Mutex
	state DeviceTraits
	*service.Thermostat
}

func NewEmulatedDevice(t *service.Thermostat, c Config) (*EmulatedDevice, error) {
	ctx := context.Background()

	// get the oauth2 token
	config := c.oauthConfig()
	token, err := c.oauthToken()
	if err != nil {
		return nil, err
	}
	source := config.TokenSource(ctx, &token)
	s, err := sdm.NewService(ctx, option.WithTokenSource(source))
	if err != nil {
		return nil, err
	}

	// list the devices
	resp, err := s.Enterprises.Devices.List("enterprises/" + c.SDMProjectID).Do()
	if err != nil {
		return nil, err
	}
	log.Println("Retrieved", len(resp.Devices), "devices")
	if len(resp.Devices) > 1 {
		log.Fatalln("Do not support multiple devices for now")
	}
	// FIXME: I'm being lazy here by only supporting one device and not checking
	// the type of the device. Works for me now.
	dn := resp.Devices[0].Name
	log.Println("Controlling device", dn)
	de := &DeviceEndpoint{
		Service: s,
		Name:    dn,
	}

	// create pubsub client and subscription
	pc, err := pubsub.NewClient(ctx, c.GCPProjectID, option.WithCredentialsFile(c.ServiceAccountKey))
	if err != nil {
		return nil, err
	}
	sub := pc.Subscription("homebridge-pubsub")

	// initialize the structure
	e := &EmulatedDevice{
		sub:            sub,
		Mutex:          &sync.Mutex{},
		DeviceEndpoint: de,
		Thermostat:     t,
	}

	// start updating the states through pubsub
	go func() {
		err := e.ListenEvents()
		if err != nil {
			log.Println(err)
		}
	}()

	// query the API once to get the initial traits
	if err := e.ForceUpdate(); err != nil {
		return nil, err
	}

	e.SetupHandlers()

	return e, nil
}

func (d *EmulatedDevice) SetupHandlers() {
	// init the thermostat service

	// set the characteristics
	// Celsius is 0, Fahrenheit is 1
	// https://developer.appld.com/documentation/homekit/hmcharacteristicvaluetemperatureunit
	// Off is 0, Heat is 1, Cool is 2, Auto is 3
	// https://developer.appld.com/documentation/homekit/hmcharacteristicvalueheatingcooling
	// Note that TargetHeatingCoolingState can be 0-3, but CurrentHeatingCoolingState
	// can only be 0-2, because "Auto" is not an actual statd.
	// https://github.com/homebridge/HAP-NodeJS/issues/815
	// Reported values must always be in Celsius
	// Another good reference of all those stuff is
	// https://github.com/brutella/hc/blob/master/gen/metadata.json
	d.TargetTemperature.OnValueRemoteGet(func() float64 {
		// depends on the set mode
		d.Lock()
		defer d.Unlock()
		return d.TargetTemp()
	})

	d.TargetTemperature.OnValueRemoteUpdate(func(n float64) {
		log.Println("Request: set target temp to", n)
		err := d.SetTargetTemp(n)
		if err != nil {
			log.Println(err)
		}
	})

	d.CurrentTemperature.OnValueRemoteGet(func() float64 {
		d.Lock()
		defer d.Unlock()
		return d.CurrentTemp()
	})

	d.TemperatureDisplayUnits.OnValueRemoteGet(func() int {
		d.Lock()
		defer d.Unlock()
		return d.DisplayUnit()
	})
	/*
		// SDM does not support changing the display unit
		d.TemperatureDisplayUnits.OnValueRemoteUpdate(func(n int) {
		})
	*/

	d.TargetHeatingCoolingState.OnValueRemoteGet(func() int {
		d.Lock()
		defer d.Unlock()
		return d.TargetMode()
	})

	d.TargetHeatingCoolingState.OnValueRemoteUpdate(func(n int) {
		log.Println("Request: set target mode to", n)
		err := d.SetTargetMode(n)
		if err != nil {
			log.Println(err)
		}
	})

	d.CurrentHeatingCoolingState.OnValueRemoteGet(func() int {
		d.Lock()
		defer d.Unlock()
		return d.CurrentHVACMode()
	})
}

func (d *EmulatedDevice) CurrentTemp() float64 {
	return d.state.CurrTemp.TempCelsius
}

func (d *EmulatedDevice) TargetTemp() float64 {
	mode := d.state.SetMode.Mode
	switch mode {
	case OFF:
		return 0
	case HEAT:
		return d.state.SetTemp.HeatCelsius
	case COOL:
		return d.state.SetTemp.CoolCelsius
	case HEATCOOL:
		return (d.state.SetTemp.HeatCelsius + d.state.SetTemp.CoolCelsius) / 2.0
	default:
		panic("unreachable set mode when querying target temp")
	}
}

func (d *EmulatedDevice) CurrentHVACMode() int {
	mode := d.state.CurrMode.Status
	switch mode {
	case OFF:
		return 0
	case "HEATING":
		return 1
	case "COOLING":
		return 2
	default:
		panic("unreachable current mode")
	}
}

func (d *EmulatedDevice) TargetMode() int {
	mode := d.state.SetMode.Mode
	switch mode {
	case OFF:
		return 0
	case HEAT:
		return 1
	case COOL:
		return 2
	case HEATCOOL:
		return 3
	default:
		panic("unreachable set mode")
	}
}

func (d *EmulatedDevice) SetTargetMode(n int) error {
	var s string
	switch n {
	case 0:
		s = OFF
	case 1:
		s = HEAT
	case 2:
		s = COOL
	case 3:
		s = HEATCOOL
	default:
		panic("unreachable target mode enumeration")
	}
	err := d.SetMode(s)
	if err != nil {
		return err
	}
	d.Lock()
	defer d.Unlock()
	log.Println("Setting target mode to", s)
	d.state.SetMode.Mode = s
	d.state.SetMode.Timestamp = time.Now()
	return nil
}

func (d *EmulatedDevice) SetTargetTemp(t float64) error {
	var err error
	switch d.state.SetMode.Mode {
	case OFF:
		err = nil
	case HEAT:
		err = d.SetHeat(t)
	case COOL:
		err = d.SetCool(t)
	case HEATCOOL:
		err = d.SetHeatCool(t-2.5, t+2.5)
	default:
		panic("unreachable target mode when setting target temp")
	}
	if err != nil {
		return err
	}
	d.Lock()
	defer d.Unlock()
	log.Println("Setting target temp to", t)
	switch d.state.SetMode.Mode {
	case HEAT:
		d.state.SetTemp.HeatCelsius = t
		d.state.SetTemp.HeatTimestamp = time.Now()
	case COOL:
		d.state.SetTemp.CoolCelsius = t
		d.state.SetTemp.CoolTimestamp = time.Now()
	case HEATCOOL:
		d.state.SetTemp.HeatCelsius = t - 2.5
		d.state.SetTemp.CoolCelsius = t + 2.5
		d.state.SetTemp.HeatTimestamp = time.Now()
		d.state.SetTemp.CoolTimestamp = time.Now()
	default:
		return nil // don't update timestamp for the OFF case
	}
	return nil
}

func (d *EmulatedDevice) DisplayUnit() int {
	unit := d.state.DisplayUnit.Unit
	switch unit {
	case "CELSIUS":
		return 0
	case "FAHRENHEIT":
		return 1
	default:
		panic("unreachable unit")
	}
}

func (d *EmulatedDevice) ListenEvents() error {
	// create a pubsub client
	ctx := context.Background()
	for {
		_ = d.sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			var update PubsubUpdate
			if err := json.Unmarshal(m.Data, &update); err != nil {
				log.Println("Error decoding pubsub update:", err)
			}
			d.UpdateTraits(update)
			m.Ack()
		})
	}
}

func (d *EmulatedDevice) ForceUpdate() error {
	log.Println("Initiating forced update")
	t := time.Now()
	r, err := d.GetDevice()
	if err != nil {
		return err
	}
	fakeUpdate := PubsubUpdate{}
	fakeUpdate.Timestamp = t
	fakeUpdate.ResourceUpdate.Traits = r
	d.UpdateTraits(fakeUpdate)
	return nil
}

func (d *EmulatedDevice) UpdateTraits(t PubsubUpdate) {
	d.Lock()
	defer d.Unlock()
	ts := t.Timestamp
	if t.ResourceUpdate.Traits.CurrMode.Status != "" && ts.After(d.state.CurrMode.Timestamp) {
		d.state.CurrMode.Status = t.ResourceUpdate.Traits.CurrMode.Status
		d.state.CurrMode.Timestamp = ts
		d.CurrentHeatingCoolingState.SetValue(d.CurrentHVACMode())
		log.Println("Current mode updated to", d.state.CurrMode.Status)
	}
	if t.ResourceUpdate.Traits.SetMode.Mode != "" && ts.After(d.state.SetMode.Timestamp) {
		d.state.SetMode.Mode = t.ResourceUpdate.Traits.SetMode.Mode
		d.state.SetMode.Timestamp = ts
		d.TargetHeatingCoolingState.SetValue(d.TargetMode())
		log.Println("Set mode updated to", d.state.SetMode.Mode)
	}
	if t.ResourceUpdate.Traits.SetTemp.HeatCelsius != 0 && ts.After(d.state.SetTemp.HeatTimestamp) {
		d.state.SetTemp.HeatCelsius = t.ResourceUpdate.Traits.SetTemp.HeatCelsius
		d.state.SetTemp.HeatTimestamp = ts
		d.TargetTemperature.SetValue(d.TargetTemp())
		log.Println("Set heat temperature updated to", d.state.SetTemp.HeatCelsius)
	}
	if t.ResourceUpdate.Traits.SetTemp.CoolCelsius != 0 && ts.After(d.state.SetTemp.CoolTimestamp) {
		d.state.SetTemp.CoolCelsius = t.ResourceUpdate.Traits.SetTemp.CoolCelsius
		d.state.SetTemp.CoolTimestamp = ts
		d.TargetTemperature.SetValue(d.TargetTemp())
		log.Println("Set cool temperature updated to", d.state.SetTemp.CoolCelsius)
	}
	if t.ResourceUpdate.Traits.CurrTemp.TempCelsius != 0 && ts.After(d.state.CurrTemp.Timestamp) {
		d.state.CurrTemp.TempCelsius = t.ResourceUpdate.Traits.CurrTemp.TempCelsius
		d.state.CurrTemp.Timestamp = ts
		d.CurrentTemperature.SetValue(d.CurrentTemp())
		log.Println("Current temperature updated to", d.state.CurrTemp.TempCelsius)
	}
	if t.ResourceUpdate.Traits.DisplayUnit.Unit != "" && ts.After(d.state.DisplayUnit.Timestamp) {
		d.state.DisplayUnit.Unit = t.ResourceUpdate.Traits.DisplayUnit.Unit
		d.state.DisplayUnit.Timestamp = ts
		d.TemperatureDisplayUnits.SetValue(d.DisplayUnit())
		log.Println("Display unit updated to", d.state.DisplayUnit.Unit)
	}
}
