package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"

	"gopkg.in/wolfeidau/lifx.v1"
)

type Bulb struct {
}

type App struct {
	bulbs map[string]*Bulb
}

func main() {
	c := lifx.NewClient()

	err := c.StartDiscovery()

	if err != nil {
		log.Fatalf("Woops %s", err)
	}

	go func() {
		for _ = range time.Tick(time.Duration(time.Second * 60)) {
			fmt.Printf("GBST see %d bulbs\n", len(c.GetBulbs()))
			for _, bulb := range c.GetBulbs() {
				c.GetBulbState(bulb)
			}
		}
	}()

	go func() {

		sub := c.Subscribe()

		for {
			event := <-sub.Events

			switch event := event.(type) {
			case *lifx.Gateway:
				log.Printf("Gateway Update %+v", event)
			case *lifx.Bulb:
				log.Printf("Bulb Update %+v", event.GetState())
			case *lifx.LightSensorState:
				log.Printf("Light Sensor Update %s %f", event.GetLifxAddress(), event.Lux)
			default:
				log.Printf("Event %+v", event)
			}

		}
	}()

	kelvin := uint16(3500)
	brightness := uint16(65535)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		kstring := r.URL.Query().Get("kelvin")
		if kstring != "" {
			kint, err := strconv.ParseInt(kstring, 10, 64)
			if err != nil {
				return
			}
			kelvin = uint16(kint)
		}
		bstring := r.URL.Query().Get("brightness")
		if bstring != "" {
			bint, err := strconv.ParseInt(bstring, 10, 64)
			if err != nil {
				return
			}
			brightness = uint16(bint)
		}
	})
	go http.ListenAndServe(":8089", nil)

	for _ = range time.Tick(time.Duration(time.Second)) {
		fmt.Printf("see %d bulbs\n", len(c.GetBulbs()))
		for _, bulb := range c.GetBulbs() {
			state := bulb.GetState()
			if state.Kelvin != kelvin || state.Brightness != brightness {
				fmt.Printf("set color/brightness on %s to (%d/%d)\n", bulb.GetLabel(), brightness, kelvin)
				c.LightColour(bulb, 0, 0, brightness, kelvin, 1000)
			}
		}
	}
}
