package main

import (
	"os"
	"sort"

	log "github.com/Sirupsen/logrus"
	"github.com/seatgeek/nomad-firehose/command/allocations"
	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "nomad-firehose"
	app.Usage = "easily firehose nomad events to a event sink"
	app.Version = "0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "log-level",
			Value:  "info",
			Usage:  "Debug level (debug, info, warn/warning, error, fatal, panic)",
			EnvVar: "LOG_LEVEL",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "allocations",
			Usage: "Firehose nomad allocation changes",
			Action: func(c *cli.Context) error {
				return allocations.NewFirehose()
			},
		},
	}
	app.Before = func(c *cli.Context) error {
		// convert the human passed log level into logrus levels
		level, err := log.ParseLevel(c.String("log-level"))
		if err != nil {
			log.Fatal(err)
		}
		log.SetLevel(level)

		return nil
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	app.Run(os.Args)
}
