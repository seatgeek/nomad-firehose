package main

import (
	"os"
	"sort"

	log "github.com/Sirupsen/logrus"
	"github.com/seatgeek/nomad-firehose/command/allocations"
	"github.com/seatgeek/nomad-firehose/command/evaluations"
	"github.com/seatgeek/nomad-firehose/command/jobs"
	"github.com/seatgeek/nomad-firehose/command/nodes"
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
				firehose, err := allocations.NewFirehose()
				if err != nil {
					return err
				}

				err = firehose.Start()
				if err != nil {
					return err
				}

				return nil
			},
		},
		{
			Name:  "nodes",
			Usage: "Firehose nomad node changes",
			Action: func(c *cli.Context) error {
				firehose, err := nodes.NewFirehose()
				if err != nil {
					return err
				}

				err = firehose.Start()
				if err != nil {
					return err
				}

				return nil
			},
		},
		{
			Name:  "evaluations",
			Usage: "Firehose nomad evaluation changes",
			Action: func(c *cli.Context) error {
				firehose, err := evaluations.NewFirehose()
				if err != nil {
					return err
				}

				err = firehose.Start()
				if err != nil {
					return err
				}

				return nil
			},
		},
		{
			Name:  "jobs",
			Usage: "Firehose nomad job changes",
			Action: func(c *cli.Context) error {
				firehose, err := jobs.NewFirehose()
				if err != nil {
					return err
				}

				err = firehose.Start()
				if err != nil {
					return err
				}

				return nil
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
		log.SetOutput(os.Stderr)

		return nil
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	app.Run(os.Args)
}
