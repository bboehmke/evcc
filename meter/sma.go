package meter

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider/sma"
	"github.com/andig/evcc/util"
	"gitlab.com/bboehmke/sunny"
)

// SMA supporting SMA Home Manager 2.0, SMA Energy Meter 30 and SMA inverter
type SMA struct {
	URI, Interface string
	Password       string `default:"0000"`
	Serial         uint32
	Power, Energy  string
	Scale          float64 `default:"1"` // power only

	log    *util.Logger
	device *sma.Device
}

func init() {
	registry.Add("sma", new(SMA))
}

//go:generate go run ../cmd/tools/decorate.go -f decorateSMA -r api.Meter -b *SMA -t "api.Battery,SoC,func() (float64, error)"

func (sm *SMA) Connect() (api.Meter, error) {
	sm.log = util.NewLogger("sma")

	if sm.Power != "" || sm.Energy != "" {
		util.NewLogger("sma").WARN.Println("energy and power setting are deprecated and will be removed in a future release")
	}

	discoverer, err := sma.GetDiscoverer(sm.Interface)
	if err != nil {
		return nil, fmt.Errorf("failed to get discoverer failed: %w", err)
	}

	switch {
	case sm.URI != "":
		sm.device, err = discoverer.DeviceByIP(sm.URI, sm.Password)
		if err != nil {
			return nil, err
		}

	case sm.Serial > 0:
		sm.device = discoverer.DeviceBySerial(sm.Serial, sm.Password)
		if sm.device == nil {
			return nil, fmt.Errorf("device not found: %d", sm.Serial)
		}

	default:
		return nil, errors.New("missing uri or serial")
	}

	// decorate api.Battery in case of inverter
	var soc func() (float64, error)
	if !sm.device.IsEnergyMeter() {
		vals, err := sm.device.GetValues()
		if err != nil {
			return nil, err
		}

		if _, ok := vals[sunny.BatteryCharge]; ok {
			soc = sm.soc
		}
	}

	return decorateSMA(sm, soc), nil
}

// CurrentPower implements the api.Meter interface
func (sm *SMA) CurrentPower() (float64, error) {
	values, err := sm.device.GetValues()
	return sm.Scale * (sma.AsFloat(values[sunny.ActivePowerPlus]) - sma.AsFloat(values[sunny.ActivePowerMinus])), err
}

// TotalEnergy implements the api.MeterEnergy interface
func (sm *SMA) TotalEnergy() (float64, error) {
	values, err := sm.device.GetValues()
	return sma.AsFloat(values[sunny.ActiveEnergyPlus]) / 3600000, err
}

// Currents implements the api.MeterCurrent interface
func (sm *SMA) Currents() (float64, float64, float64, error) {
	values, err := sm.device.GetValues()

	measurements := []sunny.ValueID{sunny.CurrentL1, sunny.CurrentL2, sunny.CurrentL3}
	var vals [3]float64
	for i := 0; i < 3; i++ {
		vals[i] = sma.AsFloat(values[measurements[i]])
	}

	return vals[0], vals[1], vals[2], err
}

// soc implements the api.Battery interface
func (sm *SMA) soc() (float64, error) {
	values, err := sm.device.GetValues()
	return sma.AsFloat(values[sunny.BatteryCharge]), err
}

// Diagnose implements the api.Diagnosis interface
func (sm *SMA) Diagnose() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "  IP:\t%s\n", sm.device.Address())
	fmt.Fprintf(w, "  Serial:\t%d\n", sm.device.SerialNumber())
	fmt.Fprintf(w, "  EnergyMeter:\t%v\n", sm.device.IsEnergyMeter())
	fmt.Fprintln(w)

	if name, err := sm.device.GetDeviceName(); err == nil {
		fmt.Fprintf(w, "  Name:\t%s\n", name)
	}

	if devClass, err := sm.device.GetDeviceClass(); err == nil {
		fmt.Fprintf(w, "  Device Class:\t0x%X\n", devClass)
	}
	fmt.Fprintln(w)

	if values, err := sm.device.GetValues(); err == nil {
		ids := make([]sunny.ValueID, 0, len(values))
		for k := range values {
			ids = append(ids, k)
		}

		sort.Slice(ids, func(i, j int) bool {
			return ids[i].String() < ids[j].String()
		})

		for _, id := range ids {
			switch values[id].(type) {
			case float64:
				fmt.Fprintf(w, "  %s:\t%f %s\n", id.String(), values[id], sunny.GetValueInfo(id).Unit)
			default:
				fmt.Fprintf(w, "  %s:\t%v %s\n", id.String(), values[id], sunny.GetValueInfo(id).Unit)
			}
		}
	}
	w.Flush()
}
