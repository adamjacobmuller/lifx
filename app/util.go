package app

import (
	"encoding/json"
	"io/ioutil"

	"github.com/gocraft/web"
)

func unmarshal_json_request(rw web.ResponseWriter, req *web.Request, models ...interface{}) error {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	for _, model := range models {
		err = json.Unmarshal(data, &model)
		if err != nil {
			return err
		}
	}

	return nil
}
