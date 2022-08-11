package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/subchen/go-log"
)

// RFLink types, based on http://www.rflink.nl/blog2/protref
var FieldTypes = map[string]string{
	"ID":         "string",    // Device ID (often a rolling code and/or device channel number)
	"AWINSP":     "hex_div10", // Average Wind speed in km. p/h
	"BARO":       "hex",       // Barometric pressure
	"BAT":        "battery",   // Battery status indicator (OK/LOW)
	"BFORECAST":  "hex",       // Weather forecast: 0=No Info/Unknown, 1=Sunny, 2=Partly Cloudy, 3=Cloudy, 4=Rain
	"CHIME":      "hex",       // Chime/Doorbell melody number
	"CMD":        "string",    // Command (ON/OFF/ALLON/ALLOFF); additional for Milight: DISCO+/DISCO-/MODE0 - MODE8
	"CO2":        "int",       // CO2 air quality
	"CURRENT":    "int",       // Current phase 1
	"CURRENT2":   "int",       // Current phase 2 (CM113)
	"CURRENT3":   "int",       // Current phase 3 (CM113)
	"DIST":       "int",       // Distance
	"HSTATUS":    "int",       // Humidity status 0=Normal, 1=Comfortable, 2=Dry, 3=Wet
	"HUM":        "int",       // Humidity (decimal value: 0-100 to indicate relative humidity in %)
	"KWATT":      "hex",       // KWatt
	"LUX":        "hex",       // Light intensity
	"METER":      "int",       // ??? Meter values (water/electricity etc.)
	"PIR":        "onoff",     // ??? ON/OFF
	"RAIN":       "hex_div10", // Total rain in mm.
	"RAINRATE":   "hex_div10", // Rain rate in mm. (per hour???)
	"RGBW":       "string",    // ??? Milight: provides 1 byte color and 1 byte brightness value
	"SET_LEVEL":  "int",       // Direct dimming level setting value (decimal value: 0-15)
	"SMOKEALERT": "onoff",     // Smoke alert ON/OFF
	"SOUND":      "int",       // ??? Noise level
	"SWITCH":     "string",    // House/Unit code like A1, P2, B16 or a button number etc.
	"TEMP":       "temp",      // Temperature celsius, high bit contains negative sign, needs division by 10 (0xC0 = 192 decimal = 19.2 degrees) (example negative temperature value: 0x80DC, high bit indicates negative temperature 0xDC=220 decimal the client side needs to divide by 10 to get -22.0 degrees
	"UV":         "hex",       // UV intensity
	"VOLT":       "int",       // ??? Voltage
	"WATT":       "int",       // ??? Watt
	"WINCHL":     "temp",      // Wind chill
	"WINDIR":     "int",       // Wind direction (integer value from 0-15) reflecting 0-360 degrees in 22.5 degree steps
	"WINGS":      "hex",       // Wind Gust in km. p/h
	"WINSP":      "hex_div10", // Wind speed in km. p/h
	"WINTMP":     "temp",      // Wind meter temperature reading
}

type Metric struct {
	LastSeen time.Time
	Gauge    prometheus.Gauge

	Vendor string
	Id     string
	Type   string

	mutex      sync.Mutex
	registered bool
}

func NewMetric(id string, sensorType string, vendor string) *Metric {
	log.Debugf("Creating new metric: vendor=%s, id=%s, type=%s", vendor, id, sensorType)

	// Lookup the friendly name from the mapping
	name := id
	if mapping != nil {
		var ok bool
		name, ok = mapping.IdNames[id]
		if !ok {
			log.Debugf("No name mapping for sensor ID %s", id)
		} else {
			log.Debugf("Mapping for sensor ID %s -> %s", id, name)
		}
	}

	m := Metric{
		LastSeen: time.Now(),
		Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "rflink",
			Name:      sensorType, // Use the type as name, eg. metrics will be rflink_temp
			ConstLabels: prometheus.Labels{
				"vendor": strings.ReplaceAll(strings.ToLower(vendor), " ", "_"),
				"id":     id,
				"type":   sensorType,
				"name":   name,
			},
		}),
		Vendor: vendor,
		Id:     id,
		Type:   sensorType,
	}

	prometheus.MustRegister(m.Gauge)
	m.registered = true

	log.Infof("Created new Gauge rflink_%s: vendor=%s, id=%s, name=%s", sensorType, vendor, id, name)

	return &m
}

func (m *Metric) Set(value float64) {
	log.Debugf("[%s|%s|%s] Locking to update value", m.Vendor, m.Id, m.Type)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.registered {
		prometheus.MustRegister(m.Gauge)
		m.registered = true
	}

	m.LastSeen = time.Now()

	m.Gauge.Set(value)
	log.Debugf("[%s|%s|%s] New value set: %.2f", m.Vendor, m.Id, m.Type, value)
}

func (m *Metric) HasExpired() bool {
	return m.LastSeen.Add(time.Second * time.Duration(*timeout)).Before(time.Now())
}

func (m *Metric) EnforceExpiration() {
	if !m.registered || !m.HasExpired() {
		return
	}

	log.Debugf("[%s|%s|%s] Locking to check expiration", m.Vendor, m.Id, m.Type)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.registered || !m.HasExpired() {
		return
	}

	prometheus.Unregister(m.Gauge)
	log.Infof("[%s|%s|%s] Unregistered metric due to timeout", m.Vendor, m.Id, m.Type)
	m.registered = false
}

func updateMetrics(msg string, m *Mapping) error {
	fields := strings.Split(msg, ";")
	if len(fields) < 3 {
		return fmt.Errorf("Malformed message: less than 3 fields")
	}

	var (
		vendor = fields[2]
		id     = ""
		values = make(map[string]float64)
	)

	// Parsing fields starting at position 3
	for i := 3; i < len(fields)-1; i++ {
		arr := strings.SplitN(fields[i], "=", 2)
		if len(arr) != 2 {
			log.Warnf("Skipping field without value at index %d: %s", i, fields[i])
			continue
		}

		kind, found := FieldTypes[arr[0]]
		if !found {
			log.Warnf("Skipping unknown field %s", arr[0])
			continue
		}

		if kind == "string" {
			if arr[0] == "ID" {
				id = strings.ToLower(arr[1])
			}
			// Any string other than the ID is ignored, there's just no way to
			// export them in prometheus
			continue
		}

		f, err := parseValue(arr[1], kind)
		if err != nil {
			log.Warnf("Skipping malformed field %s: %v", arr[0], err)
			continue
		}
		values[strings.ToLower(arr[0])] = f
	}

	vid := vendor + " " + id
	if _, ok := sensors[vid]; !ok {
		sensors[vid] = map[string]*Metric{}
	}

	// Update / create histograms for each value reported by the sensor
	for t, value := range values {
		if _, ok := sensors[vid][t]; !ok {
			sensors[vid][t] = NewMetric(id, t, vendor)
		}
		sensors[vid][t].Set(value)
	}

	return nil
}
