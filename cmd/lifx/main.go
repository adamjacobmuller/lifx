package main

import (
	log "github.com/sirupsen/logrus"
	"gitlab.adam.gs/home/lifx/app"
	"gitlab.adam.gs/home/lifx/lib"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	c := lifx.NewClient()

	err := c.StartDiscovery()
	if err != nil {
		panic(err)
	}

	a, err := app.NewApp(c)
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
