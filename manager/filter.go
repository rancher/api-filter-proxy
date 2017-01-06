package manager

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

//Filter defines the properties of a pre/post API filter
type Filter struct {
	Endpoint    string   `json:"endpoint"`
	SecretToken string   `json:"secretToken"`
	Methods     []string `json:"methods"`
	Paths       []string `json:"paths"`
}

//FilterData defines the properties of a http Request/Response Body sent to/from a filter
type FilterData struct {
	Headers map[string][]string    `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
	Status  int                    `json:"status,omitempty"`
}

func (filter *Filter) processFilter(input FilterData) (FilterData, error) {
	output := FilterData{}
	bodyContent, err := json.Marshal(input)
	if err != nil {
		return output, err
	}

	log.Debugf("Request => " + string(bodyContent))

	client := &http.Client{}
	req, err := http.NewRequest("POST", filter.Endpoint, bytes.NewBuffer(bodyContent))
	if err != nil {
		return output, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", string(len(bodyContent)))

	resp, err := client.Do(req)
	if err != nil {
		return output, err
	}
	log.Debugf("Response Status <= " + resp.Status)
	defer resp.Body.Close()

	byteContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return output, err
	}

	log.Debugf("Response <= " + string(byteContent))
	json.Unmarshal(byteContent, &output)
	output.Status = resp.StatusCode

	return output, nil
}
