package service

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
	"strings"

	"github.com/rancher/api-filter-proxy/manager"
)

var Wrapper *MuxWrapper

//MuxWrapper is a wrapper over the mux router
type MuxWrapper struct {
	Router *mux.Router
}

func (httpWrapper *MuxWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httpWrapper.Router.ServeHTTP(w, r)
}

//NewRouter creates and configures a mux router
func NewRouter(configFields manager.ConfigFileFields) *mux.Router {
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
	router.Methods("POST").Path("/v1-api-filter-proxy/reload").HandlerFunc(http.HandlerFunc(reload))
	router.NotFoundHandler = http.HandlerFunc(handleNotFoundRequest)

	return router

}
