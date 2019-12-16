package app

import (
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.adam.gs/home/lifx/lib"
)

type App struct {
	client *lifx.Client
	bulbs  map[string]*Bulb
	curves *Curves
}

func (a *App) watchOffline() {
	for _ = range time.Tick(time.Second) {
		for _, bulb := range a.bulbs {
			since := time.Since(bulb.bulb.LastSeen())
			if since > time.Hour {
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

func (a *App) SetState(bulb *lifx.Bulb) {
	addr := bulb.GetLifxAddress()
	eb, ok := a.bulbs[addr]
	if !ok {
		b := &Bulb{
			bulb:       bulb,
			app:        a,
			client:     a.client,
			Name:       bulb.GetLabel(),
			Address:    addr,
			Controlled: true,
		}
		b.setState(bulb)
		b.TargetState = bulb.GetState()
		a.bulbs[addr] = b
		log.WithFields(log.Fields{
			"address": addr,
			"name":    b.Name,
			"tags":    bulb.GetTags(),
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
					eb.ControlAfter = time.Now().Add(time.Hour)
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

func (a *App) BulbList() []*Bulb {
	var l []*Bulb
	for _, bulb := range a.bulbs {
		l = append(l, bulb)
	}
	return l
}

func (a *App) GetBulbs() []*Bulb {
	var l []*Bulb
	for _, bulb := range a.bulbs {
		l = append(l, bulb)
	}
	return l
}

func (a *App) GetLocationBulbs(location string) []*Bulb {
	var bl []*Bulb
	for _, bulb := range a.bulbs {
		if bulb.Location == location {
			bl = append(bl, bulb)
		}
	}
	return bl
}

func (a *App) GetGroupBulbs(group string) []*Bulb {
	var bl []*Bulb
	for _, bulb := range a.bulbs {
		if bulb.Group == group {
			bl = append(bl, bulb)
		}
	}
	return bl
}

func (a *App) GetLocationGroupBulbs(location string, group string) []*Bulb {
	var bl []*Bulb
	for _, bulb := range a.bulbs {
		if bulb.Location == location && bulb.Group == group {
			bl = append(bl, bulb)
		}
	}
	return bl
}

func (a *App) GetBulb(address string) *Bulb {
	for _, bulb := range a.bulbs {
		if bulb.Address == address {
			return bulb
		}
	}
	return nil
}

func (a *App) regainControl() {
	for _ = range time.Tick(time.Second) {
		for _, bulb := range a.BulbList() {
			if bulb.Controlled {
				continue
			}
			if bulb.ControlAfter.Before(time.Now()) {
				log.WithFields(log.Fields{
					"address": bulb.Address,
					"name":    bulb.Name,
					"after":   time.Since(bulb.ControlAfter),
				}).Info("regaining control of bulb")
				bulb.Controlled = true
				bulb.adjustState()
			}
		}
	}
}

func (a *App) watchAmbient() {
	for _ = range time.Tick(time.Second * 30) {
		for _, bulb := range a.BulbList() {
			a.client.GetAmbientLight(bulb.bulb)
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

	go a.regainControl()
	go a.controlState()
	go a.loadCurves()
	//go a.watchAmbient()
	RunWebServer(&a)
	return &a, nil
}
