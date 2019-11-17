package app

import (
	"fmt"

	"github.com/rancher/k3os/pkg/cli/ccapply"
	"github.com/rancher/k3os/pkg/cli/operator"
	"github.com/rancher/k3os/pkg/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	Debug bool
)

func New() *cli.App {

	app := cli.NewApp()
	app.Name = "k3os"
	app.Usage = "Booting to k3s so you don't have to"
	app.Version = version.Version
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s version %s\n", app.Name, app.Version)
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "Turn on debug logs",
			EnvVar:      "K3OS_DEBUG",
			Destination: &Debug,
		},
	}

	operatorCommand := operator.Command()
	ccapplyCommand := ccapply.Command()

	app.Commands = []cli.Command{
		operatorCommand,
		ccapplyCommand,
	}

	app.Before = func(c *cli.Context) error {
		if Debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}

	return app
}
