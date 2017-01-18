package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/rancher/api-filter-proxy/manager"
	"github.com/rancher/api-filter-proxy/model"
)

//ReturnHTTPError handles sending out CatalogError response
func ReturnHTTPError(w http.ResponseWriter, r *http.Request, httpStatus int, errorMessage string) {
	svcError := model.ProxyError{
		Status:  strconv.Itoa(httpStatus),
		Message: errorMessage,
	}
	writeError(w, svcError)
}

func writeError(w http.ResponseWriter, svcError model.ProxyError) {
	status, err := strconv.Atoi(svcError.Status)
	if err != nil {
		log.Errorf("Error writing error response %v", err)
		w.Write([]byte(svcError.Message))
		return
	}
	w.WriteHeader(status)

	jsonStr, err := json.Marshal(svcError)
	if err != nil {
		log.Errorf("Error writing error response %v", err)
		w.Write([]byte(svcError.Message))
		return
	}
	w.Write([]byte(jsonStr))
}

//Proxy is our ReverseProxy object
type Proxy struct {
	// target url of reverse proxy
	target       *url.URL
	reverseProxy *httputil.ReverseProxy
}

func NewProxy(target string) (*Proxy, error) {
	url, err := url.Parse(target)
	if err != nil {
		log.Errorf("Error reading destination URL %v", target)
		return nil, err
	}
	newProxy := httputil.NewSingleHostReverseProxy(url)
	newProxy.FlushInterval = time.Millisecond * 100
	return &Proxy{target: url, reverseProxy: newProxy}, nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	path, _ := mux.CurrentRoute(r).GetPathTemplate()

	log.Debugf("Request Path matched: %v", path)

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error reading request Body %v for path %v", r, path)
		ReturnHTTPError(w, r, http.StatusBadRequest, fmt.Sprintf("Error reading json request body, err: %v", err))
		return
	}

	var jsonInput map[string]interface{}
	if len(bodyBytes) > 0 {
		err = json.Unmarshal(bodyBytes, &jsonInput)
		if err != nil {
			log.Errorf("Error unmarshalling json request body: %v", err)
			ReturnHTTPError(w, r, http.StatusBadRequest, fmt.Sprintf("Error reading json request body: %v", err))
			return
		}
	}

	headerMap := make(map[string][]string)
	for key, value := range r.Header {
		headerMap[key] = value
	}

	api := r.URL.Path

	inputBody, inputHeaders, destination, proxyErr := manager.ProcessPreFilters(path, api, jsonInput, headerMap)
	if proxyErr.Status != "" {
		//error from some filter
		log.Debugf("Error from proxy filter %v", proxyErr)
		writeError(w, proxyErr)
		return
	}

	jsonStr, err := json.Marshal(inputBody)
	destReq, err := http.NewRequest(r.Method, r.URL.String(), bytes.NewReader(jsonStr))
	if err != nil {
		log.Errorf("Error creating new request for path %v, error: %v, body: %v", r.URL.String(), err, jsonStr)
		ReturnHTTPError(w, r, http.StatusBadRequest, fmt.Sprintf("Error creating new request for path %v to send to destination", r.URL.String()))
		return
	}
	for key, value := range inputHeaders {
		for _, singleVal := range value {
			destReq.Header.Add(key, singleVal)
		}
	}

	destProxy, err := NewProxy(destination)
	if err != nil {
		log.Errorf("Error creating a reverse proxy for destination %v", destination)
		ReturnHTTPError(w, r, http.StatusInternalServerError, fmt.Sprintf("Error creating a reverse proxy for destination %v", destination))
		return
	}
	destProxy.reverseProxy.ServeHTTP(w, destReq)
}

func handleNotFoundRequest(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Request path NOT matched to proxy config: %v, proxy to %v", r.URL.Path, manager.DefaultDestination)
	destProxy, err := NewProxy(manager.DefaultDestination)
	if err != nil {
		log.Errorf("Error creating a reverse proxy for destination %v", manager.DefaultDestination)
		ReturnHTTPError(w, r, http.StatusInternalServerError, fmt.Sprintf("Error creating a reverse proxy for destination %v", manager.DefaultDestination))
		return
	}
	destProxy.reverseProxy.ServeHTTP(w, r)
}

func reload(w http.ResponseWriter, r *http.Request) {
	log.Info("Reload proxy config")
	err := manager.Reload()
	if err != nil {
		//failed to reload the config from the config.json
		log.Debugf("Reload failed with error %v", err)
		ReturnHTTPError(w, r, http.StatusInternalServerError, "Failed to reload the proxy config")
		return
	}
	Wrapper.Router = NewRouter(manager.ConfigFields)
}
