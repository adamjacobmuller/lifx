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
	brightness = 65535
	kelvin = 4000

	defaultCurveBrightness, defaultCurveKelvin := b.app.GetDefaultCurve()
	if defaultCurveBrightness != nil {
		brightness = *defaultCurveBrightness
	}

	if defaultCurveKelvin != nil {
		kelvin = *defaultCurveKelvin
	}

	groupCurveBrightness, groupCurveKelvin := b.app.GetGroupCurve(b.Group)
	if groupCurveBrightness != nil {
		brightness = *groupCurveBrightness
	}

	if groupCurveKelvin != nil {
		kelvin = *groupCurveKelvin
	}

	if b.ManualStateUntil.After(time.Now()) {
		le := log.WithFields(log.Fields{
			"name":    b.Name,
			"address": b.Address,
			"until":   b.ManualStateUntil,
			"for":     b.ManualStateUntil.Sub(time.Now()),
		})
		if b.ManualStateKelvin != nil {
			kelvin = *b.ManualStateKelvin
			le.WithField("kelvin", kelvin)
		}
		if b.ManualStateBrightness != nil {
			brightness = *b.ManualStateBrightness
			le.WithField("brightness", brightness)
		}
		le.Debug("manually controlling state")
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
		controlAfter := time.Now().Add(time.Second * 15)
		log.WithFields(log.Fields{
			"current-brightness": state.Brightness,
			"target-brightness":  brightness,
			"current-kelvin":     state.Kelvin,
			"target-kelvin":      kelvin,
			"name":               b.Name,
			"group":              b.Group,
			"address":            b.Address,
			"control-after":      controlAfter,
		}).Info("initiating LightColor change")
		b.Controlled = false
		b.ControlAfter = controlAfter
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
