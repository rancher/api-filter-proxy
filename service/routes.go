package service

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
	"strings"

	"github.com/rancher/api-filter-proxy/manager"
)

var Router *mux.Router

//NewRouter creates and configures a mux router
func NewRouter(configFields manager.ConfigFileFields) {
	// API framework routes
	router := mux.NewRouter().StrictSlash(false)

	for _, filter := range configFields.Prefilters {
		//build router paths
		for _, path := range filter.Paths {
			for _, method := range filter.Methods {
				log.Debugf("Adding route: %v %v", strings.ToUpper(method), path)
				router.Methods(strings.ToUpper(method)).Path(path).HandlerFunc(http.HandlerFunc(handleRequest))
			}
		}
	}

	router.NotFoundHandler = http.HandlerFunc(handleNotFoundRequest)
	Router = router
}
