package sdmclient

import (
	"encoding/json"
	"fmt"
	"time"

	sdm "google.golang.org/api/smartdevicemanagement/v1"
)

type DeviceEndpoint struct {
	*sdm.Service
	Name string
}

type DeviceTraits struct {
	CurrMode struct {
		Status    string
		Timestamp time.Time `json:"-"`
	} `json:"sdm.devices.traits.ThermostatHvac"`
	CurrTemp struct {
		TempCelsius float64   `json:"ambientTemperatureCelsius"`
		Timestamp   time.Time `json:"-"`
	} `json:"sdm.devices.traits.Temperature"`
	DisplayUnit struct {
		Unit      string    `json:"temperatureScale"`
		Timestamp time.Time `json:"-"`
	} `json:"sdm.devices.traits.Settings"`
	// TODO(nateinaction): Add support for humidity
	// Humidity struct {
	// 	Percent   float64   `json:"ambientHumidityPercent"`
	// 	Timestamp time.Time `json:"-"`
	// } `json:"sdm.devices.traits.Humidity"`
	TargetMode struct {
		Mode      string
		Timestamp time.Time `json:"-"`
	} `json:"sdm.devices.traits.ThermostatMode"`
	TargetTemp struct {
		HeatCelsius   float64
		CoolCelsius   float64
		HeatTimestamp time.Time `json:"-"`
		CoolTimestamp time.Time `json:"-"`
	} `json:"sdm.devices.traits.ThermostatTemperatureSetpoint"`
}

func (d *DeviceEndpoint) GetDevice() (DeviceTraits, error) {
	res, err := d.Enterprises.Devices.Get(d.Name).Do()

	var r DeviceTraits
	if err != nil {
		return r, fmt.Errorf("failed to get device: %w", err)
	}

	if err := json.Unmarshal(res.Traits, &r); err != nil {
		return r, fmt.Errorf("failed to unmarshal device traits: %w", err)
	}

	return r, nil
}

func (d *DeviceEndpoint) SetMode(mode string) error {
	type params struct {
		Mode string `json:"mode"`
	}

	p := params{
		Mode: mode,
	}

	ep, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal mode params: %w", err)
	}

	req := &sdm.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest{
		Command: "sdm.devices.commands.ThermostatMode.SetMode",
		Params:  ep,
	}

	if _, err := d.Enterprises.Devices.ExecuteCommand(d.Name, req).Do(); err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}

	return nil
}

func (d *DeviceEndpoint) SetHeat(temp float64) error {
	type params struct {
		Temp float64 `json:"heatCelsius"`
	}

	p := params{
		Temp: temp,
	}

	ep, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal heat params: %w", err)
	}

	req := &sdm.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest{
		Command: "sdm.devices.commands.ThermostatTemperatureSetpoint.SetHeat",
		Params:  ep,
	}

	if _, err := d.Enterprises.Devices.ExecuteCommand(d.Name, req).Do(); err != nil {
		return fmt.Errorf("failed to set heat: %w", err)
	}

	return nil
}

func (d *DeviceEndpoint) SetCool(temp float64) error {
	type params struct {
		Temp float64 `json:"coolCelsius"`
	}

	p := params{
		Temp: temp,
	}

	ep, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal cool params: %w", err)
	}

	req := &sdm.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest{
		Command: "sdm.devices.commands.ThermostatTemperatureSetpoint.SetCool",
		Params:  ep,
	}

	if _, err := d.Enterprises.Devices.ExecuteCommand(d.Name, req).Do(); err != nil {
		return fmt.Errorf("failed to set cool: %w", err)
	}

	return nil
}

func (d *DeviceEndpoint) SetHeatCool(heat, cool float64) error {
	type params struct {
		Heat float64 `json:"heatCelsius"`
		Cool float64 `json:"coolCelsius"`
	}

	p := params{
		Heat: heat,
		Cool: cool,
	}

	ep, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal heatcool params: %w", err)
	}

	req := &sdm.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest{
		Command: "sdm.devices.commands.ThermostatTemperatureSetpoint.SetRange",
		Params:  ep,
	}

	if _, err := d.Enterprises.Devices.ExecuteCommand(d.Name, req).Do(); err != nil {
		return fmt.Errorf("failed to set heatcool: %w", err)
	}

	return nil
}
