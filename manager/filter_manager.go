package manager

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/rancher/api-filter-proxy/model"
)

var (
	configFile         string
	CattleURL          string
	DefaultDestination string
	ConfigFields       ConfigFileFields
	//PathPreFilters is the map storing path -> prefilters[]
	PathPreFilters map[string][]Filter
	//PathDestinations is the map storing path -> prefilters[]
	PathDestinations map[string]Destination
)

//Destination defines the properties of a Destination
type Destination struct {
	DestinationURL string   `json:"destinationURL"`
	Paths          []string `json:"paths"`
}

//ConfigFileFields stores filter config
type ConfigFileFields struct {
	Prefilters   []Filter
	Destinations []Destination
}

//SetEnv sets the parameters necessary
func SetEnv(c *cli.Context) {
	configFile = c.GlobalString("config")

	if configFile == "" {
		log.Fatal("Please specify path to the APIfilter config.json file")
		return
	}

	CattleURL = c.GlobalString("cattle-url")
	if len(CattleURL) == 0 {
		log.Fatalf("CATTLE_URL is not set")
	}

	DefaultDestination = c.GlobalString("default-destination")
	if len(DefaultDestination) == 0 {
		log.Infof("DEFAULT_DESTINATION is not set, will use CATTLE_URL as default")
		DefaultDestination = CattleURL
	}

	if configFile != "" {
		configContent, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Fatalf("Error reading config.json file at path %v", configFile)
		} else {
			ConfigFields = ConfigFileFields{}
			err = json.Unmarshal(configContent, &ConfigFields)
			if err != nil {
				log.Fatalf("config.json data format invalid, error : %v\n", err)
			}

			PathPreFilters = make(map[string][]Filter)
			for _, filter := range ConfigFields.Prefilters {
				//build the PathPreFilters map
				for _, path := range filter.Paths {
					PathPreFilters[path] = append(PathPreFilters[path], filter)
				}
			}

			PathDestinations = make(map[string]Destination)
			for _, destination := range ConfigFields.Destinations {
				//build the PathDestinations map
				for _, path := range destination.Paths {
					PathDestinations[path] = destination
				}
			}

		}
	}
}

func ProcessPreFilters(path string, body map[string]interface{}, headers map[string][]string) (map[string]interface{}, map[string][]string, string, model.ProxyError) {
	prefilters := PathPreFilters[path]
	log.Debugf("START -- Processing pre filters for request path %v", path)
	inputBody := body
	inputHeaders := headers
	for _, filter := range prefilters {
		log.Debugf("-- Processing pre filter %v for request path %v --", filter, path)

		requestData := FilterData{}
		requestData.Body = inputBody
		requestData.Headers = inputHeaders

		responseData, err := filter.processFilter(requestData)
		if err != nil {
			log.Errorf("Error %v processing the filter %v", err, filter)
			svcErr := model.ProxyError{
				Status:  strconv.Itoa(http.StatusInternalServerError),
				Message: fmt.Sprintf("Error %v processing the filter %v", err, filter),
			}
			return inputBody, inputHeaders, "", svcErr
		}
		if responseData.Status == 200 {
			if responseData.Body != nil {
				inputBody = responseData.Body
			}
			if responseData.Headers != nil {
				inputHeaders = responseData.Headers
			}
		} else {
			//error
			log.Errorf("Error response %v - %v while processing the filter %v", responseData.Status, responseData.Body, filter)
			svcErr := model.ProxyError{
				Status:  strconv.Itoa(responseData.Status),
				Message: fmt.Sprintf("Error response while processing the filter %v", filter.Endpoint),
			}

			return inputBody, inputHeaders, "", svcErr
		}
	}

	//send the final body and headers to destination
	destination, ok := PathDestinations[path]
	destinationURL := destination.DestinationURL
	if !ok {
		destinationURL = DefaultDestination
	}
	log.Debugf("DONE -- Processing pre filters for request path %v, following to destination %v", path, destinationURL)

	return inputBody, inputHeaders, destinationURL, model.ProxyError{}
}
