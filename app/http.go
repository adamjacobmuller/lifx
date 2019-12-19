package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gocraft/web"
	log "github.com/sirupsen/logrus"
)

type BulbJSON struct {
	Name          string    `json:"name"`
	Address       string    `json:"address"`
	Lux           float32   `json:"lux,omitempty"`
	Location      string    `json:"location,omitempty"`
	Group         string    `json:"group,omitempty"`
	LastSeen      time.Time `json:"last-seen"`
	LastSeenSince string    `json:"last-seen-since"`
	Hue           int       `json:"hue"`
	Saturation    int       `json:"saturation"`
	Brightness    int       `json:"brightness"`
	Kelvin        int       `json:"kelvin"`
	Dim           int       `json:"dim"`
	Power         int       `json:"power"`
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

	router.Get("/curves", (*Context).ListCurves)
	router.Get("/bulbs", (*Context).ListBulbs)
	router.Get("/bulbs/:*", (*Context).ListBulbs)
	router.Post("/bulbs/:*", (*Context).UpdateBulbs)
	router.Delete("/bulbs/:*", (*Context).ReleaseBulbs)
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

func ParseUpdateBulbRequest(ur *UpdateBulbRequest) (*time.Time, *time.Duration, *uint16, *uint16, error) {
	if ur.Until != nil && ur.Duration != nil {
		return nil, nil, nil, nil, errors.New("don't set both until and duration")
	}
	var until *time.Time
	var duration *time.Duration
	var brightness *uint16
	var kelvin *uint16

	if ur.Until != nil {
		until = ur.Until
	} else if ur.Duration != nil {
		dv, err := time.ParseDuration(*ur.Duration)
		if err != nil {
			return nil, nil, nil, nil, errors.New("can't parse duration")
		}
		duration = &dv
		uv := time.Now().Add(dv)
		until = &uv
	} else {
		return nil, nil, nil, nil, errors.New("must set until or duration")
	}
	if ur.Brightness == nil && ur.Kelvin == nil {
		return nil, nil, nil, nil, errors.New("must set one of brightness or kelvin")
	}

	if ur.Brightness != nil {
		bv := uint16(*ur.Brightness)
		brightness = &bv
	}

	if ur.Kelvin != nil {
		kv := uint16(*ur.Kelvin)
		kelvin = &kv
	}

	return until, duration, brightness, kelvin, nil
}

func (c *Context) ReleaseBulbs(rw web.ResponseWriter, req *web.Request) {
	bulbs, err := c.filter(req.PathParams["*"])
	if err != nil {
		http.Error(rw, err.Error(), 400)
	}

	for _, bulb := range bulbs {
		le := log.WithFields(log.Fields{
			"address": bulb.Address,
			"name":    bulb.Name,
		})
		bulb.ManualStateUntil = time.Now()
		bulb.ManualStateBrightness = nil
		bulb.ManualStateKelvin = nil
		bulb.Controlled = true
		le.Info("releasing bulb from manual control")
	}
}

func (c *Context) UpdateBulbs(rw web.ResponseWriter, req *web.Request) {
	bulbs, err := c.filter(req.PathParams["*"])
	if err != nil {
		http.Error(rw, err.Error(), 400)
	}

	ur := &UpdateBulbRequest{}
	err = unmarshal_json_request(rw, req, ur)
	if err != nil {
		panic(err)
	}

	until, duration, brightness, kelvin, err := ParseUpdateBulbRequest(ur)
	if err != nil {
		http.Error(rw, err.Error(), 400)
	}

	for _, bulb := range bulbs {
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
		if brightness != nil {
			bulb.ManualStateBrightness = brightness
			le = le.WithField("brightness", brightness)
		}
		if kelvin != nil {
			bulb.ManualStateKelvin = kelvin
			le = le.WithField("kelvin", kelvin)
		}
		le.Info("setting bulb to manual control")
	}
}

func (c *Context) UpdateBulb(rw web.ResponseWriter, req *web.Request) {
	ur := &UpdateBulbRequest{}
	err := unmarshal_json_request(rw, req, ur)
	if err != nil {
		panic(err)
	}

	until, duration, brightness, kelvin, err := ParseUpdateBulbRequest(ur)
	if err != nil {
		http.Error(rw, err.Error(), 400)
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
	if brightness != nil {
		bulb.ManualStateBrightness = brightness
		le = le.WithField("brightness", brightness)
	}
	if kelvin != nil {
		bulb.ManualStateKelvin = kelvin
		le = le.WithField("kelvin", kelvin)
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
				Location:      bulb.Location,
				Group:         bulb.Group,
				Lux:           bulb.Lux,
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

func (c *Context) ListCurves(rw web.ResponseWriter, req *web.Request) {
	d, err := json.Marshal(c.App.curves)
	if err != nil {
		panic(err)
	}
	rw.Header().Add("content-type", "application/json")
	rw.Write(d)
}

func (c *Context) ListBulbs(rw web.ResponseWriter, req *web.Request) {
	var v []*BulbJSON
	for _, bulb := range c.App.BulbList() {
		state := bulb.bulb.GetState()
		v = append(v, &BulbJSON{
			Name:          bulb.Name,
			Address:       bulb.Address,
			Location:      bulb.Location,
			Group:         bulb.Group,
			Lux:           bulb.Lux,
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
