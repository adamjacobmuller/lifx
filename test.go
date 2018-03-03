package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wolfeidau/lifx"
)

type Bulb struct {
	client            *lifx.Client
	bulb              *lifx.Bulb
	app               *App
	Name              string
	Address           string
	Online            bool
	LastChange        time.Time
	LastStateUpdate   time.Time
	LastState         lifx.BulbState
	Controlled        bool
	UncontrolledSince time.Time
	TargetState       lifx.BulbState
}

// {Hue:0 Saturation:0 Brightness:16449 Kelvin:2500 Dim:0 Power:65535 Visible:true}

type App struct {
	client *lifx.Client
	bulbs  map[string]*Bulb
}

func (a *App) watchOffline() {
	for _ = range time.Tick(time.Second) {
		for _, bulb := range a.bulbs {
			since := time.Since(bulb.bulb.LastSeen())
			if since > time.Second*20 {
				if bulb.Online {
					bulb.Online = false
					log.WithFields(log.Fields{
						"name":   bulb.Name,
						"addrss": bulb.Address,
						"since":  since,
					}).Info("bulb is now offline")
				}
			} else {
				bulb.Online = true
			}
		}
	}
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
		brightness = 2000
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
		fallthrough
	case 14: // 2PM
		fallthrough
	case 15: // 3PM
		fallthrough
	case 16: // 4PM
		brightness = 32768
		kelvin = 4000
	case 17: // 5PM
		fallthrough
	case 18: // 6PM
		fallthrough
	case 19: // 7PM
		fallthrough
	case 20: // 8PM
		brightness = 8192
		kelvin = 3500
	case 21: // 9PM
		fallthrough
	case 22: // 10PM
		fallthrough
	case 23: // 11PM
		brightness = 4096
		kelvin = 3000
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
			"brightness": brightness,
			"kelvin":     kelvin,
			"name":       b.Name,
		}).Info("initiating LightColor change")
		b.Controlled = false
		b.UncontrolledSince = time.Now()
		b.TargetState.Kelvin = kelvin
		b.TargetState.Brightness = brightness
		b.client.LightColour(b.bulb, hue, sat, brightness, kelvin, timing)
	}
}

func (b *Bulb) setState(bulb *lifx.Bulb) {
	state := bulb.GetState()
	b.LastStateUpdate = time.Now()
	b.LastState = state
}

func (b *Bulb) targetedChange(bulb *lifx.Bulb) ([]string, bool) {
	state := bulb.GetState()
	var differences []string
	var targeted bool
	targeted = true
	if b.TargetState.Hue != state.Hue {
		differences = append(differences, fmt.Sprintf("hue %d->%d", b.LastState.Power, state.Power))
		targeted = false
	}
	if b.TargetState.Saturation != state.Saturation {
		differences = append(differences, fmt.Sprintf("saturation %d->%d", b.LastState.Saturation, state.Saturation))
		targeted = false
	}
	if b.TargetState.Brightness != state.Brightness {
		differences = append(differences, fmt.Sprintf("brightness %d->%d", b.LastState.Brightness, state.Brightness))
		targeted = false
	}
	if b.TargetState.Kelvin != state.Kelvin {
		differences = append(differences, fmt.Sprintf("kelvin %d->%d", b.LastState.Kelvin, state.Kelvin))
		targeted = false
	}
	if b.TargetState.Dim != state.Dim {
		differences = append(differences, fmt.Sprintf("dim %d->%d", b.LastState.Dim, state.Dim))
		targeted = false
	}
	if b.TargetState.Power != state.Power {
		differences = append(differences, fmt.Sprintf("power %d->%d", b.LastState.Power, state.Power))
		targeted = false
	}
	return differences, targeted
}

func (b *Bulb) changed(bulb *lifx.Bulb) ([]string, bool) {
	state := bulb.GetState()
	var changes []string
	var changed bool
	if b.LastState.Hue != state.Hue {
		changes = append(changes, fmt.Sprintf("hue %d->%d", b.LastState.Power, state.Power))
		changed = true
	}
	if b.LastState.Saturation != state.Saturation {
		changes = append(changes, fmt.Sprintf("saturation %d->%d", b.LastState.Saturation, state.Saturation))
		changed = true
	}
	if b.LastState.Brightness != state.Brightness {
		changes = append(changes, fmt.Sprintf("brightness %d->%d", b.LastState.Brightness, state.Brightness))
		changed = true
	}
	if b.LastState.Kelvin != state.Kelvin {
		changes = append(changes, fmt.Sprintf("kelvin %d->%d", b.LastState.Kelvin, state.Kelvin))
		changed = true
	}
	if b.LastState.Dim != state.Dim {
		changed = true
		changes = append(changes, fmt.Sprintf("dim %d->%d", b.LastState.Dim, state.Dim))
	}
	if b.LastState.Power != state.Power {
		changed = true
		changes = append(changes, fmt.Sprintf("power %d->%d", b.LastState.Power, state.Power))
	}
	return changes, changed
}

func (a *App) SetState(bulb *lifx.Bulb) {
	addr := bulb.GetLifxAddress()
	eb, ok := a.bulbs[addr]
	if !ok {
		b := &Bulb{
			bulb:              bulb,
			app:               a,
			client:            a.client,
			Name:              bulb.GetLabel(),
			Address:           addr,
			Controlled:        true,
			UncontrolledSince: time.Now(),
		}
		b.setState(bulb)
		b.TargetState = bulb.GetState()
		a.bulbs[addr] = b
		log.WithFields(log.Fields{
			"address": addr,
			"name":    b.Name,
		}).Info("new bulb")
	} else {
		since := time.Since(eb.bulb.LastSeen())
		changes, changed := eb.changed(bulb)
		if changed {
			log.WithFields(log.Fields{
				"address":    addr,
				"lastupdate": since,
				"name":       eb.Name,
				"changes":    changes,
			}).Info("state changed!")
			eb.LastChange = time.Now()
			if eb.Controlled {
				targetMismatch, targeted := eb.targetedChange(bulb)
				if targeted {
					eb.Controlled = true
				} else {
					log.WithFields(log.Fields{
						"address":        addr,
						"lastupdate":     since,
						"name":           eb.Name,
						"targetMismatch": targetMismatch,
					}).Info("target mismatched, relinquishing control")
					eb.Controlled = false
					eb.UncontrolledSince = time.Now()
				}
			} else {
				_, targeted := eb.targetedChange(bulb)
				if targeted {
					log.WithFields(log.Fields{
						"address":    addr,
						"lastupdate": since,
						"name":       eb.Name,
					}).Info("target acquired, regaining control")
					eb.Controlled = true
				}
			}
		}
		if eb.Online == false {
			sinceLastUpdate := time.Since(eb.LastStateUpdate)
			log.WithFields(log.Fields{
				"address": addr,
				"offline": sinceLastUpdate,
				"name":    eb.Name,
			}).Debug("bulb is back online")
		}
		eb.setState(bulb)
	}
}

type BulbJSON struct {
	Name          string    `json:"name"`
	LastSeen      time.Time `json:"last-seen"`
	LastSeenSince string    `json:"last-seen-since"`
	Hue           int       `json:"hue"`
	Saturation    int       `json:"saturation"`
	Brightness    int       `json:"brightness"`
	Kelvin        int       `json:"kelvin"`
	Dim           int       `json:"dim"`
	Power         int       `json:"power"`

	//Luminosity    int       `json:"luminosity"`
}

func (a *App) BulbList() []*Bulb {
	var l []*Bulb
	for _, bulb := range a.bulbs {
		l = append(l, bulb)
	}
	return l
}

func (a *App) BulbListJSON() []BulbJSON {
	var v []BulbJSON
	for _, bulb := range a.BulbList() {
		state := bulb.bulb.GetState()
		v = append(v, BulbJSON{
			Name:          bulb.Name,
			LastSeen:      bulb.bulb.LastSeen(),
			LastSeenSince: time.Since(bulb.bulb.LastSeen()).String(),
			Hue:           int(state.Hue),
			Saturation:    int(state.Saturation),
			Brightness:    int(state.Brightness),
			Kelvin:        int(state.Kelvin),
			Dim:           int(state.Dim),
			Power:         int(state.Power),
		})
	}
	return v
}

func (a *App) Handle(w http.ResponseWriter, r *http.Request) {
	d, err := json.Marshal(a.BulbListJSON())
	if err != nil {
		panic(err)
	}
	w.Header().Add("content-type", "application/json")
	w.Write(d)
}

func (a *App) regainControl() {
	for _ = range time.Tick(time.Second) {
		for _, bulb := range a.BulbList() {
			if bulb.Controlled {
				continue
			}
			if bulb.UncontrolledSince.Add(time.Second * 30).Before(time.Now()) {
				log.WithFields(log.Fields{
					"address": bulb.Address,
					"name":    bulb.Name,
					"after":   time.Since(bulb.UncontrolledSince),
				}).Info("regaining control of bulb")
				bulb.Controlled = true
				bulb.adjustState()
			}
		}
	}
}

func (a *App) controlState() {
	for _ = range time.Tick(time.Second) {
		for _, bulb := range a.BulbList() {
			if !bulb.Controlled {
				continue
			}
			bulb.adjustState()
		}
	}
}

func NewApp(c *lifx.Client) (*App, error) {
	a := App{
		bulbs:  make(map[string]*Bulb),
		client: c,
	}
	http.HandleFunc("/", a.Handle)
	go http.ListenAndServe(":8089", nil)
	//go a.watchOffline()
	go a.regainControl()
	go a.controlState()
	return &a, nil
}

func main() {
	c := lifx.NewClient()

	err := c.StartDiscovery()
	if err != nil {
		panic(err)
	}

	a, err := NewApp(c)
	if err != nil {
		panic(err)
	}

	sub := c.Subscribe()

	for {
		event := <-sub.Events

		switch event := event.(type) {
		case *lifx.Gateway:
			//log.Printf("Gateway Update %+v", event)
		case *lifx.Bulb:
			//log.Printf("Bulb Update %+v", event.GetState())
			a.SetState(event)
		case *lifx.LightSensorState:
			//log.Printf("Light Sensor Update %s %f", event.GetLifxAddress(), event.Lux)
		default:
			log.Printf("Event %+v", event)
		}

	}
}
