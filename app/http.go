package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gocraft/web"
	log "github.com/sirupsen/logrus"
)

type BulbJSON struct {
	Name          string    `json:"name"`
	Address       string    `json:"address"`
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

type Context struct {
	App *App
}

func RunWebServer(a *App) {
	router := web.New(Context{})

	router.Middleware(func(ctx *Context, rw web.ResponseWriter,
		req *web.Request, next web.NextMiddlewareFunc) {
		ctx.App = a
		next(rw, req)
	})

	router.Get("/bulbs", (*Context).ListBulbs)
	router.Get("/bulb/:bulb_id", (*Context).GetBulb)
	router.Post("/bulb/:bulb_id", (*Context).UpdateBulb)

	go http.ListenAndServe(":8089", router)
	log.Info("listening")
}

type UpdateBulbRequest struct {
	Until      *time.Time `json:"until,omitempty"`
	Duration   *string    `json:"duration,omitempty"`
	Brightness *int       `json:"brightness,omitempty"`
	Kelvin     *int       `json:"kelvin,omitempty"`
}

func (c *Context) UpdateBulb(rw web.ResponseWriter, req *web.Request) {
	ur := &UpdateBulbRequest{}
	err := unmarshal_json_request(rw, req, ur)
	if err != nil {
		panic(err)
	}

	if ur.Until == nil && ur.Duration == nil {
		http.Error(rw, "must set until or duration", 400)
		return
	}
	if ur.Until != nil && ur.Duration != nil {
		http.Error(rw, "don't set both until and duration", 400)
		return
	}
	var until *time.Time
	var duration *time.Duration
	if ur.Until != nil {
		until = ur.Until
	} else if ur.Duration != nil {
		dv, err := time.ParseDuration(*ur.Duration)
		if err != nil {
			http.Error(rw, "can't parse duration", 400)
			return
		}
		duration = &dv
		uv := time.Now().Add(dv)
		until = &uv
	} else {
		http.Error(rw, "must set until or duration", 400)
		return
	}
	if ur.Brightness == nil && ur.Kelvin == nil {
		http.Error(rw, "must set one of brightness or duration", 400)
		return
	}

	bulb := c.App.GetBulb(req.PathParams["bulb_id"])
	if bulb == nil {
		http.Error(rw, "no such bulb", 404)
		return
	}
	le := log.WithFields(log.Fields{
		"address": bulb.Address,
		"name":    bulb.Name,
	})
	bulb.ManualStateUntil = *until
	le = le.WithField("until", ur.Until)
	if duration != nil {
		le = le.WithField("duration", duration)
	}
	bulb.ManualStateBrightness = nil
	bulb.ManualStateKelvin = nil
	if ur.Brightness != nil {
		brightness := uint16(*ur.Brightness)
		bulb.ManualStateBrightness = &brightness
		le = le.WithField("brightness", ur.Brightness)
	}
	if ur.Kelvin != nil {
		kelvin := uint16(*ur.Kelvin)
		bulb.ManualStateKelvin = &kelvin
		le = le.WithField("kelvin", ur.Kelvin)
	}
	le.Info("setting bulb to manual control")
}

func (c *Context) GetBulb(rw web.ResponseWriter, req *web.Request) {
	for _, bulb := range c.App.BulbList() {
		if bulb.Address == req.PathParams["bulb_id"] {
			state := bulb.bulb.GetState()
			v := &BulbJSON{
				Name:          bulb.Name,
				Address:       bulb.Address,
				LastSeen:      bulb.bulb.LastSeen(),
				LastSeenSince: time.Since(bulb.bulb.LastSeen()).String(),
				Hue:           int(state.Hue),
				Saturation:    int(state.Saturation),
				Brightness:    int(state.Brightness),
				Kelvin:        int(state.Kelvin),
				Dim:           int(state.Dim),
				Power:         int(state.Power),
			}
			d, err := json.Marshal(v)
			if err != nil {
				panic(err)
			}
			rw.Header().Add("content-type", "application/json")
			rw.Write(d)
			return
		}
	}
	http.Error(rw, "no such bulb", 404)
}

func (c *Context) ListBulbs(rw web.ResponseWriter, req *web.Request) {
	var v []*BulbJSON
	for _, bulb := range c.App.BulbList() {
		state := bulb.bulb.GetState()
		v = append(v, &BulbJSON{
			Name:          bulb.Name,
			Address:       bulb.Address,
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
	d, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	rw.Header().Add("content-type", "application/json")
	rw.Write(d)
}
