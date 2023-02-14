package emulation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/brutella/hap/service"
	"github.com/yangl1996/nesthub/internal/config"
	"github.com/yangl1996/nesthub/pkg/sdmclient"
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
		Traits sdmclient.DeviceTraits
	}
}

type EmulatedDevice struct {
	*sdmclient.DeviceEndpoint
	sub *pubsub.Subscription
	*sync.Mutex
	state sdmclient.DeviceTraits
	*service.Thermostat
}

func NewEmulatedDevice(ctx context.Context, c *config.Config, t *service.Thermostat) (*EmulatedDevice, error) {
	// Setup sdm service
	tokenSource, err := c.NewOAuthTokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth token source: %w", err)
	}

	s, err := sdm.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create sdm service: %w", err)
	}

	// list the devices
	resp := ListDevicesWithRetries(s, c)

	log.Println("Retrieved", len(resp.Devices), "devices")

	// TODO: support multiple devices
	if len(resp.Devices) > 1 {
		log.Fatalf("nesthub only supports one device, more than one device found: %v", resp.Devices)
	}
	// TODO: I'm being lazy here by only supporting one device and not checking
	// the type of the device. Works for me now.
	dn := resp.Devices[0].Name

	log.Println("Controlling device", dn)

	de := &sdmclient.DeviceEndpoint{
		Service: s,
		Name:    dn,
	}

	// create pubsub client and subscription
	pc, err := pubsub.NewClient(ctx, c.GCPProjectID, option.WithCredentialsFile(c.ServiceAccountKey))
	if err != nil {
		return nil, err
	}

	// initialize the structure
	e := &EmulatedDevice{
		sub:            pc.Subscription("homebridge-pubsub"),
		Mutex:          &sync.Mutex{},
		DeviceEndpoint: de,
		Thermostat:     t,
	}

	// start updating the states through pubsub
	go func() {
		if err := e.ListenEvents(); err != nil {
			log.Printf("pubsub event listener encountered an error: %v", err)
		}
	}()

	// query the API once to get the initial traits
	if err := e.ForceUpdate(); err != nil {
		return nil, fmt.Errorf("failed to force update device: %w", err)
	}

	e.SetupHandlers()

	return e, nil
}

func ListDevicesWithRetries(s *sdm.Service, c *config.Config) *sdm.GoogleHomeEnterpriseSdmV1ListDevicesResponse {
	delay := 1
	delayMultiplier := 2
	delayMax := 120

	for {
		resp, err := s.Enterprises.Devices.List("enterprises/" + c.SDMProjectID).Do()
		if err != nil {
			delayDuration := time.Duration(delay) * time.Second
			log.Printf("Failed to connect to SDM API, retrying in %s: %v", delayDuration, err)
			time.Sleep(delayDuration)

			delay *= delayMultiplier
			if delay > delayMax {
				delay = delayMax
			}

			continue
		}

		return resp
	}
}

func (d *EmulatedDevice) SetupHandlers() {
	// init the thermostat service
	//
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
	// https://github.com/brutella/hap/blob/master/gen/metadata.json
	d.TargetTemperature.ValueRequestFunc = func(*http.Request) (interface{}, int) {
		// depends on the set mode
		d.Lock()
		defer d.Unlock()

		temp := d.TargetTemp()

		return temp, 0
	}

	d.TargetTemperature.OnValueRemoteUpdate(func(n float64) {
		// TODO: Set temp in hap as well as SDM to prevent HomeKit delay?
		if err := d.SetTargetTemp(n); err != nil {
			log.Println("HomeKit: Error updating target temperature:", err)
			return
		}

		log.Println("HomeKit: Target temperature updated to", n)
	})

	d.CurrentTemperature.ValueRequestFunc = func(*http.Request) (interface{}, int) {
		d.Lock()
		defer d.Unlock()

		temp := d.CurrentTemp()

		return temp, 0
	}

	d.TemperatureDisplayUnits.ValueRequestFunc = func(*http.Request) (interface{}, int) {
		d.Lock()
		defer d.Unlock()

		unit := d.DisplayUnit()

		return unit, 0
	}

	/*
		// SDM does not support changing the display unit
		d.TemperatureDisplayUnits.OnValueRemoteUpdate(func(n int) {
		})
	*/

	d.TargetHeatingCoolingState.ValueRequestFunc = func(*http.Request) (interface{}, int) {
		d.Lock()
		defer d.Unlock()

		mode := d.TargetMode()

		return mode, 0
	}

	d.TargetHeatingCoolingState.OnValueRemoteUpdate(func(n int) {
		// TODO: Set mode in hap as well as SDM to prevent HomeKit delay?
		if err := d.SetTargetMode(n); err != nil {
			log.Println("HomeKit: Error updating target mode:", err)
			return
		}

		log.Println("HomeKit: Target mode updated to", n)
	})

	d.CurrentHeatingCoolingState.ValueRequestFunc = func(*http.Request) (interface{}, int) {
		d.Lock()
		defer d.Unlock()

		mode := d.CurrentMode()

		return mode, 0
	}
}

func (d *EmulatedDevice) CurrentTemp() float64 {
	return d.state.CurrTemp.TempCelsius
}

func (d *EmulatedDevice) TargetTemp() float64 {
	mode := d.state.TargetMode.Mode
	switch mode {
	case OFF:
		return 0
	case HEAT:
		return d.state.TargetTemp.HeatCelsius
	case COOL:
		return d.state.TargetTemp.CoolCelsius
	case HEATCOOL:
		return (d.state.TargetTemp.HeatCelsius + d.state.TargetTemp.CoolCelsius) / 2.0
	default:
		panic("unreachable set mode when querying target temp")
	}
}

func (d *EmulatedDevice) CurrentMode() int {
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
	mode := d.state.TargetMode.Mode
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
	switch n {
	case 0:
		return d.SetMode(OFF)
	case 1:
		return d.SetMode(HEAT)
	case 2:
		return d.SetMode(COOL)
	case 3:
		return d.SetMode(HEATCOOL)
	default:
		// TODO handle error
		panic("unreachable target mode enumeration")
	}
}

func (d *EmulatedDevice) SetTargetTemp(t float64) error {
	switch d.state.TargetMode.Mode {
	case OFF:
		return nil // don't update timestamp for the OFF case
	case COOL:
		return d.SetCool(t)
	case HEAT:
		return d.SetHeat(t)
	case HEATCOOL:
		return d.SetHeatCool(t-2.5, t+2.5)
	default:
		// TODO handle error
		panic("unreachable target mode when setting target temp")
	}
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
				m.Nack()
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
	if sDiff(t.ResourceUpdate.Traits.CurrMode.Status, d.state.CurrMode.Status) && ts.After(d.state.CurrMode.Timestamp) {
		d.state.CurrMode.Status = t.ResourceUpdate.Traits.CurrMode.Status
		d.state.CurrMode.Timestamp = ts

		if err := d.CurrentHeatingCoolingState.SetValue(d.CurrentMode()); err != nil {
			log.Println("Nest: Error updating current mode:", err)
			return
		}

		log.Println("Nest: Current mode updated to", d.state.CurrMode.Status)
	}

	if fDiff(t.ResourceUpdate.Traits.CurrTemp.TempCelsius, d.state.CurrTemp.TempCelsius) && ts.After(d.state.CurrTemp.Timestamp) {
		d.state.CurrTemp.TempCelsius = t.ResourceUpdate.Traits.CurrTemp.TempCelsius
		d.state.CurrTemp.Timestamp = ts

		d.CurrentTemperature.SetValue(d.CurrentTemp())

		log.Println("Nest: Current temperature updated to", d.state.CurrTemp.TempCelsius)
	}

	if sDiff(t.ResourceUpdate.Traits.DisplayUnit.Unit, d.state.DisplayUnit.Unit) && ts.After(d.state.DisplayUnit.Timestamp) {
		d.state.DisplayUnit.Unit = t.ResourceUpdate.Traits.DisplayUnit.Unit
		d.state.DisplayUnit.Timestamp = ts

		if err := d.TemperatureDisplayUnits.SetValue(d.DisplayUnit()); err != nil {
			log.Println("Nest: Error updating display units:", err)
			return
		}

		log.Println("Nest: Display unit updated to", d.state.DisplayUnit.Unit)
	}

	// TODO(nateinaction): Add support for humidity
	// if t.ResourceUpdate.Traits.Humidity.Percent != 0 && ts.After(d.state.Humidity.Timestamp) {
	// 	d.state.Humidity.Percent = t.ResourceUpdate.Traits.Humidity.Percent
	// 	d.state.Humidity.Timestamp = ts
	// 	d.CurrentRelativeHumidity.SetValue(d.Humidity())
	// 	log.Println("Humidity updated to", d.state.Humidity.Percent)
	// }

	if sDiff(t.ResourceUpdate.Traits.TargetMode.Mode, d.state.TargetMode.Mode) && ts.After(d.state.TargetMode.Timestamp) {
		d.state.TargetMode.Mode = t.ResourceUpdate.Traits.TargetMode.Mode
		d.state.TargetMode.Timestamp = ts

		if err := d.TargetHeatingCoolingState.SetValue(d.TargetMode()); err != nil {
			log.Println("Nest: Error updating target mode:", err)
			return
		}

		log.Println("Nest: Target mode updated to", d.state.TargetMode.Mode)
	}

	if fDiff(t.ResourceUpdate.Traits.TargetTemp.CoolCelsius, d.state.TargetTemp.CoolCelsius) && ts.After(d.state.TargetTemp.CoolTimestamp) {
		d.state.TargetTemp.CoolCelsius = t.ResourceUpdate.Traits.TargetTemp.CoolCelsius
		d.state.TargetTemp.CoolTimestamp = ts

		d.TargetTemperature.SetValue(d.TargetTemp())

		log.Println("Nest: Target cool temperature updated to", d.state.TargetTemp.CoolCelsius)
	}

	if fDiff(t.ResourceUpdate.Traits.TargetTemp.HeatCelsius, d.state.TargetTemp.HeatCelsius) && ts.After(d.state.TargetTemp.HeatTimestamp) {
		d.state.TargetTemp.HeatCelsius = t.ResourceUpdate.Traits.TargetTemp.HeatCelsius
		d.state.TargetTemp.HeatTimestamp = ts

		d.TargetTemperature.SetValue(d.TargetTemp())

		log.Println("Nest: Target heat temperature updated to", d.state.TargetTemp.HeatCelsius)
	}
}

// fDiff returns true if the floats are different and the new float is non-zero
func fDiff(new, old float64) bool {
	return new != 0 && new != old
}

// sDiff returns true if the strings are different and the new string is non-empty
func sDiff(new, old string) bool {
	return new != "" && new != old
}
