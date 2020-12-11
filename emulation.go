package main

import (
	"cloud.google.com/go/pubsub"
	"golang.org/x/oauth2"
	sdm "google.golang.org/api/smartdevicemanagement/v1"
	"google.golang.org/api/option"
	"context"
	"encoding/json"
	"log"
	"time"
	"sync"
)

type PubsubUpdate struct {
	Timestamp time.Time
	ResourceUpdate struct {
		Traits DeviceTraits
	}
}

type EmulatedDevice struct {
	*DeviceEndpoint
	sub *pubsub.Subscription
	*sync.Mutex
	state DeviceTraits
}


func NewEmulatedDevice(c Config) (*EmulatedDevice, error) {
	ctx := context.Background()

	// get the oauth2 token
	config := oauth2.Config {
		ClientID: c.OAuthClientID,
		ClientSecret: c.OAuthClientSecret,
		Endpoint: oauth2.Endpoint {
			TokenURL: "https://oauth2.googleapis.com/token",
			AuthURL: "https://accounts.google.com/o/oauth2/auth",
		},
		RedirectURL: "https://www.google.com",
	}
	token := oauth2.Token {
		AccessToken: c.AccessToken,
		TokenType: "Bearer",
		RefreshToken: c.RefreshToken,
		Expiry: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	source := config.TokenSource(ctx, &token)
	s, err := sdm.NewService(ctx, option.WithTokenSource(source))
	if err != nil {
		return nil, err
	}

	// list the devices
	resp, err := s.Enterprises.Devices.List("enterprises/"+c.SDMProjectID).Do()
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
	de := &DeviceEndpoint {
		Service: s,
		Name: dn,
	}

	// create pubsub client and subscription
	pc, err := pubsub.NewClient(ctx, c.GCPProjectID, option.WithCredentialsFile(c.ServiceAccountKey))
	if err != nil {
		return nil, err
	}
	sub := pc.Subscription(c.PubSubID)

	// initialize the structure
	e := &EmulatedDevice{
		sub: sub,
		Mutex: &sync.Mutex{},
		DeviceEndpoint: de,
	}

	// start updating the states through pubsub
	go e.ListenEvents()

	// query the API once to get the initial traits
	err = e.ForceUpdate()
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (d *EmulatedDevice) CurrentTemp() float64 {
	d.Lock()
	defer d.Unlock()
	return d.state.CurrTemp.TempCelsius
}

func (d *EmulatedDevice) TargetTemp() float64 {
	d.Lock()
	defer d.Unlock()
	mode := d.state.SetMode.Mode
	switch mode {
		case "OFF": return 0
		case "HEAT": return d.state.SetTemp.HeatCelsius
		case "COOL": return d.state.SetTemp.CoolCelsius
		case "HEATCOOL": return (d.state.SetTemp.HeatCelsius + d.state.SetTemp.CoolCelsius) / 2.0
		default: panic("unreachable set mode when querying target temp")
	}
}

func (d *EmulatedDevice) CurrentHVACMode() int {
	d.Lock()
	defer d.Unlock()
	mode := d.state.CurrMode.Status
	switch mode {
		case "OFF": return 0
		case "HEATING": return 1
		case "COOLING": return 2
		default: panic("unreachable current mode")
	}
}

func (d *EmulatedDevice) TargetMode() int {
	d.Lock()
	defer d.Unlock()
	mode := d.state.SetMode.Mode
	switch mode {
		case "OFF": return 0
		case "HEAT": return 1
		case "COOL": return 2
		case "HEATCOOL": return 3
		default: panic("unreachable set mode")
	}
}

func (d *EmulatedDevice) SetTargetMode(n int) error {
	var s string
	switch n {
		case 0: s = "OFF"
		case 1: s = "HEAT"
		case 2: s = "COOL"
		case 3: s = "HEATCOOL"
		default: panic("unreachable target mode enumeration")
	}
	err := d.SetMode(s)
	if err != nil {
		return err
	}
	d.Lock()
	defer d.Unlock()
	d.state.SetMode.Mode = s
	d.state.SetMode.Timestamp = time.Now()
	return nil
}

func (d *EmulatedDevice) SetTargetTemp(t float64) error {
	var err error
	switch d.state.SetMode.Mode {
		case "OFF": err = nil
		case "HEAT": err = d.SetHeat(t)
		case "COOL": err = d.SetCool(t)
		case "HEATCOOL": err = d.SetHeatCool(t-2.5, t+2.5)
		default: panic("unreachable target mode when setting target temp")
	}
	if err != nil {
		return err
	}
	d.Lock()
	defer d.Unlock()
	switch d.state.SetMode.Mode {
		case "HEAT":
			d.state.SetTemp.HeatCelsius = t
			d.state.SetTemp.HeatTimestamp = time.Now()
		case "COOL":
			d.state.SetTemp.CoolCelsius = t
			d.state.SetTemp.CoolTimestamp = time.Now()
		case "HEATCOOL":
			d.state.SetTemp.HeatCelsius = t - 2.5
			d.state.SetTemp.CoolCelsius = t + 2.5
			d.state.SetTemp.HeatTimestamp = time.Now()
			d.state.SetTemp.CoolTimestamp = time.Now()
		default: return nil	// don't update timestamp for the OFF case
	}
	return nil
}

func (d *EmulatedDevice) DisplayUnit() int {
	d.Lock()
	defer d.Unlock()
	unit := d.state.DisplayUnit.Unit
	switch unit {
		case "CELSIUS": return 0
		case "FAHRENHEIT": return 1
		default: panic("unreachable unit")
	}
}

func (d *EmulatedDevice) ListenEvents() error {
	// create a pubsub client
	ctx := context.Background()
	for ;; {
		_ = d.sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			var update PubsubUpdate
			json.Unmarshal(m.Data, &update)
			d.UpdateTraits(update)
			m.Ack()
		})
	}
	return nil
}

func (d *EmulatedDevice) ForceUpdate() error {
	log.Println("Initiating forced update")
	t := time.Now()
	r, err := d.GetDevice()
	if err != nil {
		return err
	}
	fakeUpdate := PubsubUpdate {}
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
		log.Println("Current mode updated to", d.state.CurrMode.Status)
	}
	if t.ResourceUpdate.Traits.SetMode.Mode != "" && ts.After(d.state.SetMode.Timestamp) {
		d.state.SetMode.Mode = t.ResourceUpdate.Traits.SetMode.Mode
		d.state.SetMode.Timestamp = ts
		log.Println("Set mode updated to", d.state.SetMode.Mode)
	}
	if t.ResourceUpdate.Traits.SetTemp.HeatCelsius != 0 && ts.After(d.state.SetTemp.HeatTimestamp) {
		d.state.SetTemp.HeatCelsius = t.ResourceUpdate.Traits.SetTemp.HeatCelsius
		d.state.SetTemp.HeatTimestamp = ts
		log.Println("Set heat temperature updated to", d.state.SetTemp.HeatCelsius)
	}
	if t.ResourceUpdate.Traits.SetTemp.CoolCelsius != 0 && ts.After(d.state.SetTemp.CoolTimestamp) {
		d.state.SetTemp.CoolCelsius = t.ResourceUpdate.Traits.SetTemp.CoolCelsius
		d.state.SetTemp.CoolTimestamp = ts
		log.Println("Set cool temperature updated to", d.state.SetTemp.CoolCelsius)
	}
	if t.ResourceUpdate.Traits.CurrTemp.TempCelsius != 0 && ts.After(d.state.CurrTemp.Timestamp) {
		d.state.CurrTemp.TempCelsius = t.ResourceUpdate.Traits.CurrTemp.TempCelsius
		d.state.CurrTemp.Timestamp = ts
		log.Println("Current temperature updated to", d.state.CurrTemp.TempCelsius)
	}
	if t.ResourceUpdate.Traits.DisplayUnit.Unit != "" && ts.After(d.state.DisplayUnit.Timestamp) {
		d.state.DisplayUnit.Unit = t.ResourceUpdate.Traits.DisplayUnit.Unit
		d.state.DisplayUnit.Timestamp = ts
		log.Println("Display unit updated to", d.state.DisplayUnit.Unit)
	}
}
