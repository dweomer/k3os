package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rancher/k3os/pkg/cc"
	"github.com/rancher/k3os/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	initrd      = false
	bootPhase   = false
	configPhase = false
	dump        = false
	dumpJSON    = false
)

// Command `ccapply`
func Command() cli.Command {
	return command // value copy on purpose
}

var command = cli.Command{
	Name:      "config",
	Usage:     "configure k3OS",
	ShortName: "cfg",
	Aliases: []string{
		"ccapply",
	},
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:        "initrd",
			Destination: &initrd,
			Usage:       "Run initrd stage",
		},
		cli.BoolFlag{
			Name:        "boot",
			Destination: &bootPhase,
			Usage:       "Run boot stage",
		},
		cli.BoolFlag{
			Name:        "config",
			Destination: &configPhase,
			Usage:       "Run os-config stage",
		},
		cli.BoolFlag{
			Name:        "dump",
			Destination: &dump,
			Usage:       "Print current configuration",
		},
		cli.BoolFlag{
			Name:        "dump-json",
			Destination: &dumpJSON,
			Usage:       "Print current configuration in json",
		},
	},
	Before: func(c *cli.Context) error {
		if os.Getuid() != 0 {
			return fmt.Errorf("must be run as root")
		}
		return nil
	},
	Action: func(*cli.Context) {
		if err := Main(); err != nil {
			logrus.Error(err)
		}
	},
}

// Main `ccapply`
func Main() error {
	cfg, err := config.ReadConfig()
	if err != nil {
		return err
	}

	if initrd {
		return cc.InitApply(&cfg)
	} else if bootPhase {
		return cc.BootApply(&cfg)
	} else if configPhase {
		return cc.ConfigApply(&cfg)
	} else if dump {
		return config.Write(cfg, os.Stdout)
	} else if dumpJSON {
		return json.NewEncoder(os.Stdout).Encode(&cfg)
	}

	return cc.RunApply(&cfg)
}
