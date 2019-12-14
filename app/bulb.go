package app

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.adam.gs/home/lifx/lib"
)

type Bulb struct {
	client          *lifx.Client
	bulb            *lifx.Bulb
	app             *App
	Name            string
	Address         string
	Online          bool
	LastChange      time.Time
	LastStateUpdate time.Time
	LastState       lifx.BulbState
	Controlled      bool
	ControlAfter    time.Time
	TargetState     lifx.BulbState
	Lux             float32
	Location        string
	Group           string

	ManualStateKelvin     *uint16
	ManualStateBrightness *uint16
	ManualStateUntil      time.Time
}

func bulbDiff(left lifx.BulbState, right lifx.BulbState) ([]string, bool) {
	var changed bool = false
	var differences []string
	if left.Hue != right.Hue {
		changed = true
		differences = append(differences, fmt.Sprintf("hue %d->%d", left.Hue, right.Hue))
	}
	if left.Saturation != right.Saturation {
		changed = true
		differences = append(differences, fmt.Sprintf("saturation %d->%d", left.Saturation, right.Saturation))
	}
	if left.Brightness != right.Brightness {
		changed = true
		differences = append(differences, fmt.Sprintf("brightness %d->%d", left.Brightness, right.Brightness))
	}
	if left.Kelvin != right.Kelvin {
		changed = true
		differences = append(differences, fmt.Sprintf("kelvin %d->%d", left.Kelvin, right.Kelvin))
	}
	if left.Dim != right.Dim {
		changed = true
		differences = append(differences, fmt.Sprintf("dim %d->%d", left.Dim, right.Dim))
	}
	if left.Power != right.Power {
		changed = true
		differences = append(differences, fmt.Sprintf("power %d->%d", left.Power, right.Power))
	}
	return differences, changed
}

func (b *Bulb) adjustState() {
	var hue uint16
	var sat uint16
	var brightness uint16
	var kelvin uint16
	var timing uint32
	timing = 10000
	brightness = 2500
	kelvin = 2500
	hour := time.Now().Hour()
	switch hour {
	case 0:
		fallthrough
	case 1:
		fallthrough
	case 2:
		fallthrough
	case 3:
		fallthrough
	case 4:
		brightness = 2048
		kelvin = 2500
	case 5:
		fallthrough
	case 6:
		fallthrough
	case 7:
		fallthrough
	case 8:
		fallthrough
	case 9:
		fallthrough
	case 10:
		fallthrough
	case 11:
		fallthrough
	case 12:
		fallthrough
	case 13: // 1PM
		brightness = 16384
		kelvin = 5000
	case 14: // 2PM
		fallthrough
	case 15: // 3PM
		fallthrough
	case 16: // 4PM
		brightness = 16384
		kelvin = 4000
	case 17: // 5PM
		fallthrough
	case 18: // 6PM
		brightness = 16384
		kelvin = 3750
	case 19: // 7PM
		fallthrough
	case 20: // 8PM
		kelvin = 3500
		if b.Address == "d073d522a994" || b.Address == "d073d5228b34" {
			brightness = 16384
		} else {
			brightness = 8192
		}
	case 21: // 9PM
		fallthrough
	case 22: // 10PM
		brightness = 8192
		kelvin = 3000
	case 23: // 11PM
		brightness = 4096
		kelvin = 2500
	}

	if b.ManualStateUntil.After(time.Now()) {
		le := log.WithFields(log.Fields{
			"name":    b.Name,
			"address": b.Address,
		})
		if b.ManualStateKelvin != nil {
			kelvin = *b.ManualStateKelvin
			le.WithField("kelvin", kelvin)
		}
		if b.ManualStateBrightness != nil {
			brightness = *b.ManualStateBrightness
			le.WithField("brightness", brightness)
		}
		le.Info("manually controlling state")
	} else {
		log.WithFields(log.Fields{
			"name":             b.Name,
			"address":          b.Address,
			"ManualStateUntil": b.ManualStateUntil,
		}).Debug("not manually controlling state")
	}

	state := b.bulb.GetState()
	var update bool = false
	if state.Brightness != brightness {
		update = true
	}
	if state.Kelvin != kelvin {
		update = true
	}
	if update {
		log.WithFields(log.Fields{
			"current-brightness": state.Brightness,
			"target-brightness":  brightness,
			"current-kelvin":     state.Kelvin,
			"target-kelvin":      kelvin,
			"name":               b.Name,
			"address":            b.Address,
		}).Info("initiating LightColor change")
		b.Controlled = false
		b.ControlAfter = time.Now().Add(time.Second * 15)
		b.TargetState.Kelvin = kelvin
		b.TargetState.Brightness = brightness
		b.client.LightColour(b.bulb, hue, sat, brightness, kelvin, timing)
	}
}

func (b *Bulb) setState(bulb *lifx.Bulb) {
	state := bulb.GetState()
	b.LastStateUpdate = time.Now()
	b.LastState = state
	b.Lux = bulb.GetLux()
	b.Location = bulb.GetLocation()
	b.Group = bulb.GetGroup()
}

func (b *Bulb) targetedChange(bulb *lifx.Bulb) ([]string, bool) {
	state := bulb.GetState()
	differences, changed := bulbDiff(b.TargetState, state)
	return differences, !changed
}

func (b *Bulb) changed(bulb *lifx.Bulb) ([]string, bool) {
	state := bulb.GetState()
	return bulbDiff(b.LastState, state)
}
