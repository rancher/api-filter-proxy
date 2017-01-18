package main

import (
	"fmt"
	"github.com/rancher/api-filter-proxy/manager"
	"github.com/rancher/api-filter-proxy/service"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

var VERSION = "v0.0.0-dev"

func beforeApp(c *cli.Context) error {
	if c.GlobalBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "api-filter-proxy"
	app.Version = VERSION
	app.Usage = "Rancher api-filter-proxy supporting api specific filters"
	app.Author = "Rancher Labs, Inc."
	app.Email = ""
	app.Before = beforeApp
	app.Action = StartService
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "config",
			Usage: fmt.Sprintf(
				"Specify path to the config.json file containg API Filter configuration",
			),
		},
		cli.StringFlag{
			Name: "default-destination",
			Usage: fmt.Sprintf(
				"Specify the default destination URL for proxying any request paths not specified in config.json",
			),
			EnvVar: "DEFAULT_DESTINATION",
		},
		cli.StringFlag{
			Name: "cattle-url",
			Usage: fmt.Sprintf(
				"Specify Cattle endpoint URL",
			),
			EnvVar: "CATTLE_URL",
		},
		cli.StringFlag{
			Name: "cattle-access-key",
			Usage: fmt.Sprintf(
				"Specify Cattle access key",
			),
			EnvVar: "CATTLE_ACCESS_KEY",
		},
		cli.StringFlag{
			Name: "cattle-secret-key",
			Usage: fmt.Sprintf(
				"Specify Cattle secret key",
			),
			EnvVar: "CATTLE_SECRET_KEY",
		},
		cli.BoolFlag{
			Name: "debug",
			Usage: fmt.Sprintf(
				"Set true to get debug logs",
			),
		},
		cli.StringFlag{
			Name:  "listen",
			Value: ":8091",
			Usage: fmt.Sprintf(
				"Address to listen to (TCP)",
			),
		},
	}

	app.Run(os.Args)
}

func StartService(c *cli.Context) {
	if c.GlobalBool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	textFormatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(textFormatter)

	manager.SetEnv(c)

	log.Info("Starting Rancher api-filter-proxy service %v", manager.ConfigFields)

	router := service.NewRouter(manager.ConfigFields)
	service.Wrapper = &service.MuxWrapper{router}

	log.Info("Listening on ", c.GlobalString("listen"))

	log.Fatal(http.ListenAndServe(c.GlobalString("listen"), service.Wrapper))

}
