package manager

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/rancher/api-filter-proxy/filters"
	//to register all filters
	_ "github.com/rancher/api-filter-proxy/filters/http"
	"github.com/rancher/api-filter-proxy/model"
	"github.com/rancher/api-filter-proxy/util"
	"github.com/urfave/cli"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var (
	configFile         string
	CattleURL          string
	DefaultDestination string
	ConfigFields       ConfigFileFields
	//PathPreFilters is the map storing path -> prefilters[]
	PathPreFilters map[string][]model.FilterData
	//PathDestinations is the map storing path -> prefilters[]
	PathDestinations  map[string]Destination
	refreshReqChannel *chan int
)

//Destination defines the properties of a Destination
type Destination struct {
	DestinationURL string   `json:"destinationURL"`
	Paths          []string `json:"paths"`
}

//ConfigFileFields stores filter config
type ConfigFileFields struct {
	Prefilters   []model.FilterData
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

	refChan := make(chan int, 1)
	refreshReqChannel = &refChan

	if configFile != "" {
		ConfigFields = ConfigFileFields{}
		PathPreFilters = make(map[string][]model.FilterData)
		PathDestinations = make(map[string]Destination)
		err := Reload()
		if err != nil {
			log.Fatalf("Failed to load the proxy Config: %v", err)
		}
	}
}

func Reload() error {
	//put msg on channel, so that any other request can wait
	select {
	case *refreshReqChannel <- 1:
		if configFile != "" {
			configContent, err := ioutil.ReadFile(configFile)
			if err != nil {
				log.Errorf("Error reading config.json file at path %v", configFile)
				<-*refreshReqChannel
				return fmt.Errorf("Error reading config.json file at path %v", configFile)
			}
			updatedConfigFields := ConfigFileFields{}
			err = json.Unmarshal(configContent, &updatedConfigFields)
			if err != nil {
				log.Errorf("config.json data format invalid, error : %v\n", err)
				<-*refreshReqChannel
				return fmt.Errorf("Proxy config.json data format invalid, error : %v", err)
			}

			updatedPathPreFilters := make(map[string][]model.FilterData)
			for _, filter := range updatedConfigFields.Prefilters {
				//build the PathPreFilters map
				for _, path := range filter.Paths {
					updatedPathPreFilters[path] = append(updatedPathPreFilters[path], filter)
				}
			}

			updatedPathDestinations := make(map[string]Destination)
			for _, destination := range updatedConfigFields.Destinations {
				//build the PathDestinations map
				for _, path := range destination.Paths {
					updatedPathDestinations[path] = destination
				}
			}
			ConfigFields = updatedConfigFields
			PathPreFilters = updatedPathPreFilters
			PathDestinations = updatedPathDestinations

		}
		<-*refreshReqChannel
	default:
		log.Infof("Reload config is already in process, skipping")
	}
	return nil
}

func ProcessPreFilters(path string, api string, body map[string]interface{}, headers map[string][]string) (map[string]interface{}, map[string][]string, string, model.ProxyError) {
	prefilters := PathPreFilters[path]
	log.Debugf("START -- Processing pre filters for request path %v", path)
	inputBody := body
	inputHeaders := headers
	//add uuid
	UUID := util.GenerateUUID()
	//envId
	envID := extractEnvID(api)

	for _, filterData := range prefilters {
		log.Debugf("-- Processing pre filter %v for request path %v --", filterData, path)

		requestData := model.APIRequestData{}
		requestData.Body = inputBody
		requestData.Headers = inputHeaders
		requestData.UUID = UUID
		requestData.APIPath = api
		if envID != "" {
			requestData.EnvID = envID
		}

		apiFilter := filters.GetAPIFilter(filterData.Name)
		responseData, err := apiFilter.ProcessFilter(filterData, requestData)
		if err != nil {
			log.Errorf("Error %v processing the filter %v", err, filterData)
			svcErr := model.ProxyError{
				Status:  strconv.Itoa(http.StatusInternalServerError),
				Message: fmt.Sprintf("Error %v processing the filter %v", err, filterData),
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
			log.Errorf("Error response %v - %v while processing the filter %v", responseData.Status, responseData.Body, filterData)
			svcErr := model.ProxyError{
				Status:  strconv.Itoa(responseData.Status),
				Message: fmt.Sprintf("Error response while processing the filter %v", filterData.Endpoint),
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

func extractEnvID(requestURL string) string {
	envID := ""
	if strings.Contains(requestURL, "/projects/") {
		parts := strings.Split(requestURL, "/projects/")
		if len(parts) > 1 {
			subParts := strings.Split(parts[1], "/")
			envID = subParts[0]
		}
	}
	return envID
}
