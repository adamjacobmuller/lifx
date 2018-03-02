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
	Bulb            *lifx.Bulb
	Name            string
	Address         string
	Changeable      bool
	Online          bool
	LastChange      time.Time
	LastStateUpdate time.Time
	LastState       lifx.BulbState
}

// {Hue:0 Saturation:0 Brightness:16449 Kelvin:2500 Dim:0 Power:65535 Visible:true}

type App struct {
	bulbs map[string]*Bulb
}

func (a *App) watchOffline() {
	for _ = range time.Tick(time.Second) {
		for _, bulb := range a.bulbs {
			since := time.Since(bulb.Bulb.LastSeen())
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

func (b *Bulb) setState(bulb *lifx.Bulb) {
	state := bulb.GetState()
	b.LastStateUpdate = time.Now()
	b.LastState = state
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
			Bulb:    bulb,
			Name:    bulb.GetLabel(),
			Address: addr,
		}
		b.setState(bulb)
		a.bulbs[addr] = b
		log.WithFields(log.Fields{
			"address": addr,
			"name":    b.Name,
		}).Info("new bulb")
	} else {
		since := time.Since(eb.Bulb.LastSeen())
		if eb.Online == false {
			sinceLastUpdate := time.Since(eb.LastStateUpdate)
			log.WithFields(log.Fields{
				"address": addr,
				"offline": sinceLastUpdate,
				"name":    eb.Name,
			}).Info("bulb is back online")
			eb.setState(bulb)
		} else {
			changes, changed := eb.changed(bulb)
			if changed {
				log.WithFields(log.Fields{
					"address":    addr,
					"lastupdate": since,
					"name":       eb.Name,
					"changes":    changes,
				}).Info("state changed!")
				eb.LastChange = time.Now()
				eb.setState(bulb)
			}
		}
	}
}

type BulbJSON struct {
	Name          string    `json:"name"`
	LastSeen      time.Time `json:"last-seen"`
	LastSeenSince string    `json:"last-seen-since"`
}

func (a *App) BulbList() []BulbJSON {
	var v []BulbJSON
	for _, bulb := range a.bulbs {
		v = append(v, BulbJSON{
			Name:          bulb.Name,
			LastSeen:      bulb.Bulb.LastSeen(),
			LastSeenSince: time.Since(bulb.Bulb.LastSeen()).String(),
		})
	}
	return v
}

func (a *App) Handle(w http.ResponseWriter, r *http.Request) {
	d, err := json.Marshal(a.BulbList())
	if err != nil {
		panic(err)
	}
	w.Header().Add("content-type", "application/json")
	w.Write(d)
}

func NewApp() (*App, error) {
	a := App{
		bulbs: make(map[string]*Bulb),
	}
	http.HandleFunc("/", a.Handle)
	go http.ListenAndServe(":8089", nil)
	//go a.watchOffline()
	return &a, nil
}

func main() {
	c := lifx.NewClient()

	err := c.StartDiscovery()
	if err != nil {
		panic(err)
	}

	a, err := NewApp()
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
