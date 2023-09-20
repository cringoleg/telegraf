package regime_calculator

import (
	"encoding/json"
	"io"
	"os"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
	"github.com/tidwall/gjson"
)

type RegimeCalculator struct {
	ConfigPath string `toml:"config_path"`
	RegimeMap  map[int]string
	Log        telegraf.Logger `toml:"-"`
}

type RegimeConfig struct {
	Regimes []struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Devices []struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Values []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"values"`
		} `json:"devices"`
	} `json:"regimes"`
}

func (r *RegimeCalculator) SampleConfig() string {
	return `
  ## Path to the JSON configuration file
  config_path = "/path/to/config.json"
`
}

func (r *RegimeCalculator) Description() string {
	return "Calculate regime based on regimeId and rename measurement"
}

func (r *RegimeCalculator) Init() error {
	// Load the JSON file here
	jsonFile, err := os.Open(r.ConfigPath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var config RegimeConfig
	json.Unmarshal(byteValue, &config)

	r.RegimeMap = make(map[int]string)
	for _, regime := range config.Regimes {
		r.RegimeMap[regime.ID] = regime.Name
	}

	return nil
}

func (r *RegimeCalculator) Apply(in ...telegraf.Metric) []telegraf.Metric {
	for _, metric := range in {
		rawLog, ok := metric.GetField("json_log")
		if ok {
			regimeID := gjson.Get(rawLog.(string), "data.devices.common.regimeID").Int()
			if !ok {
				r.Log.Error("regimeID not found")
				continue // Skip this metric if regimeId is not present or not a number
			}

			for i := 0; i < 32; i++ { // Assuming 32-bit value
				if int(regimeID)>>i&1 == 1 {
					regimeName, exists := r.RegimeMap[i+1] // Assuming regime IDs start from 1
					if exists {
						metric.AddTag(regimeName, "enabled")
					}
				}
			}
		}
	}
	return in
}

func init() {
	processors.Add("regime_calculator", func() telegraf.Processor {
		return &RegimeCalculator{}
	})
}
