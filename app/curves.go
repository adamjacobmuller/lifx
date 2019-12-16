package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"
)

type Curves struct {
	Default *Curve            `json:"default"`
	Groups  map[string]*Curve `json:"groups"`
}

type Curve struct {
	Groups []string             `json:"groups"`
	Hours  map[string]CurveHour `json:"hours"`
}
type CurveHour struct {
	Brightness *uint16 `json:"brightness,omitempty"`
	Kelvin     *uint16 `json:"kelvin,omitempty"`
}

func (a *App) GetDefaultCurve() (*uint16, *uint16) {
	if a.curves == nil {
		return nil, nil
	}

	if a.curves.Default == nil {
		return nil, nil
	}

	hour := fmt.Sprintf("%d", time.Now().Hour())

	curve, ok := a.curves.Default.Hours[hour]
	if !ok {
		return nil, nil
	}

	return curve.Brightness, curve.Kelvin
}

func (a *App) GetGroupCurve(group string) (*uint16, *uint16) {
	if a.curves == nil {
		return nil, nil
	}

	if a.curves.Groups == nil {
		return nil, nil
	}

	groupCurves, ok := a.curves.Groups[group]
	if !ok {
		return nil, nil
	}

	hour := fmt.Sprintf("%d", time.Now().Hour())

	curve, ok := groupCurves.Hours[hour]
	if !ok {
		return nil, nil
	}

	return curve.Brightness, curve.Kelvin
}

func (a *App) loadCurves() error {
	curves := &Curves{}
	curves.Groups = make(map[string]*Curve)

	defaultCurve, err := loadCurve("/home/adam/curves/default.json")
	if err != nil {
		return err
	}

	curves.Default = defaultCurve

	groupCurveFiles, err := filepath.Glob("/home/adam/curves/groups/*.json")
	if err != nil {
		return err
	}

	for _, groupCurveFile := range groupCurveFiles {
		groupCurve, err := loadCurve(groupCurveFile)
		if err != nil {
			return err
		}

		for _, groupCurveName := range groupCurve.Groups {
			curves.Groups[groupCurveName] = groupCurve
		}
	}

	a.curves = curves

	return nil
}

func loadCurve(filename string) (*Curve, error) {
	curveData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	curve := &Curve{}

	err = json.Unmarshal(curveData, curve)
	if err != nil {
		return nil, err
	}

	return curve, nil
}
