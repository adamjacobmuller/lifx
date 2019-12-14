package app

import (
	"errors"
	"log"
	"regexp"
	"strings"
)

func (c *Context) filter(filter string) ([]*Bulb, error) {
	log.Printf("filter is %s", filter)
	filterRegexp := regexp.MustCompile("^([^!=<>~]+)(?:(!=|>|<|=|~)(.*))?$")

	filterItems := strings.Split(filter, ",")
	var group string
	var location string
	for _, filterItem := range filterItems {
		filterItemParts := filterRegexp.FindStringSubmatch(filterItem)
		if len(filterItemParts) == 0 {
			return nil, errors.New("invalid filter expression (regexp does not match)")
		}
		if filterItemParts[2] != "=" {
			return nil, errors.New("can't filter on anything but equals")
		}
		if filterItemParts[1] == "group" {
			group = filterItemParts[3]
		} else if filterItemParts[1] == "location" {
			location = filterItemParts[3]
		} else {
			return nil, errors.New("can only filter on group or location")
		}
	}

	if group != "" && location != "" {
		return c.App.GetLocationGroupBulbs(location, group), nil
	} else if group != "" {
		return c.App.GetGroupBulbs(group), nil
	} else if location != "" {
		return c.App.GetLocationBulbs(location), nil
	} else {
		return c.App.GetBulbs(), nil
	}
}
