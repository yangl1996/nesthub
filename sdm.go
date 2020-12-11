package main

import (
	sdm "google.golang.org/api/smartdevicemanagement/v1"
	"encoding/json"
	"time"
)

type DeviceEndpoint struct {
	*sdm.Service
	Name string
}

type DeviceTraits struct {
	CurrMode struct {
		Status string
		Timestamp time.Time `json:"-"`
	} `json:"sdm.devices.traits.ThermostatHvac"`
	SetMode struct {
		Mode string
		Timestamp time.Time `json:"-"`
	} `json:"sdm.devices.traits.ThermostatMode"`
	SetTemp struct {
		HeatCelsius float64
		CoolCelsius float64
		HeatTimestamp time.Time `json:"-"`
		CoolTimestamp time.Time `json:"-"`
	} `json:"sdm.devices.traits.ThermostatTemperatureSetpoint"`
	CurrTemp struct {
		TempCelsius float64 `json:"ambientTemperatureCelsius"`
		Timestamp time.Time `json:"-"`
	} `json:"sdm.devices.traits.Temperature"`
	DisplayUnit struct {
		Unit string `json:"temperatureScale"`
		Timestamp time.Time `json:"-"`
	} `json:"sdm.devices.traits.Settings"`
}

func (d *DeviceEndpoint) GetDevice() (DeviceTraits, error) {
	res, err := d.Enterprises.Devices.Get(d.Name).Do()
	var r DeviceTraits
	if err != nil {
		return r, err
	}

	err = json.Unmarshal(res.Traits, &r)
	if err != nil {
		return r, err
	}

	return r, nil
}

func (d *DeviceEndpoint) SetMode(mode string) error {
	type params struct {
		Mode string `json:"mode"`
	}
	p := params {
		Mode: mode,
	}
	ep, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req := &sdm.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest {
		Command: "sdm.devices.commands.ThermostatMode.SetMode",
		Params: ep,
	}
	_, err = d.Enterprises.Devices.ExecuteCommand(d.Name, req).Do()
	if err != nil {
		return err
	}
	return nil
}

func (d *DeviceEndpoint) SetHeat(temp float64) error {
	type params struct {
		Temp float64 `json:"heatCelsius"`
	}
	p := params {
		Temp: temp,
	}
	ep, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req := &sdm.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest {
		Command: "sdm.devices.commands.ThermostatTemperatureSetpoint.SetHeat",
		Params: ep,
	}
	_, err = d.Enterprises.Devices.ExecuteCommand(d.Name, req).Do()
	if err != nil {
		return err
	}
	return nil
}

func (d *DeviceEndpoint) SetCool(temp float64) error {
	type params struct {
		Temp float64 `json:"coolCelsius"`
	}
	p := params {
		Temp: temp,
	}
	ep, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req := &sdm.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest {
		Command: "sdm.devices.commands.ThermostatTemperatureSetpoint.SetCool",
		Params: ep,
	}
	_, err = d.Enterprises.Devices.ExecuteCommand(d.Name, req).Do()
	if err != nil {
		return err
	}
	return nil
}

func (d *DeviceEndpoint) SetHeatCool(heat, cool float64) error {
	type params struct {
		Heat float64 `json:"heatCelsius"`
		Cool float64 `json:"coolCelsius"`
	}
	p := params {
		Heat: heat,
		Cool: cool,
	}
	ep, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req := &sdm.GoogleHomeEnterpriseSdmV1ExecuteDeviceCommandRequest {
		Command: "sdm.devices.commands.ThermostatTemperatureSetpoint.SetRange",
		Params: ep,
	}
	_, err = d.Enterprises.Devices.ExecuteCommand(d.Name, req).Do()
	if err != nil {
		return err
	}
	return nil
}
