package main

import (
	"os"
	"sort"

	gelf "github.com/seatgeek/logrus-gelf-formatter"
	"github.com/seatgeek/nomad-firehose/command/allocations"
	"github.com/seatgeek/nomad-firehose/command/deployments"
	"github.com/seatgeek/nomad-firehose/command/evaluations"
	"github.com/seatgeek/nomad-firehose/command/jobs"
	"github.com/seatgeek/nomad-firehose/command/nodes"
	"github.com/seatgeek/nomad-firehose/helper"
	log "github.com/sirupsen/logrus"
	cli "gopkg.in/urfave/cli.v1"
)

var GitCommit string

func main() {
	app := cli.NewApp()
	app.Name = "nomad-firehose"
	app.Usage = "easily firehose nomad events to a event sink"

	app.Version = GitCommit

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "log-level",
			Value:  "info",
			Usage:  "Debug level (debug, info, warn/warning, error, fatal, panic)",
			EnvVar: "LOG_LEVEL",
		},
		cli.StringFlag{
			Name:   "log-format",
			Value:  "text",
			Usage:  "json or text",
			EnvVar: "LOG_FORMAT",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "allocations",
			Usage: "Firehose nomad allocation changes",
			Action: func(c *cli.Context) error {
				firehose, err := allocations.NewFirehose()
				if err != nil {
					log.Fatal(err)
				}

				manager := helper.NewManager(firehose)
				if err := manager.Start(); err != nil {
					log.Fatal(err)
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
					log.Fatal(err)
				}

				manager := helper.NewManager(firehose)
				if err := manager.Start(); err != nil {
					log.Fatal(err)
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
					log.Fatal(err)
				}

				manager := helper.NewManager(firehose)
				if err := manager.Start(); err != nil {
					log.Fatal(err)
				}

				return nil
			},
		},
		{
			Name:  "jobs",
			Usage: "Firehose nomad job changes",
			Action: func(c *cli.Context) error {
				firehose, err := jobs.NewJobFirehose()
				if err != nil {
					log.Fatal(err)
				}

				manager := helper.NewManager(firehose)
				if err := manager.Start(); err != nil {
					log.Fatal(err)
				}

				return nil
			},
		},
		{
			Name:  "jobliststubs",
			Usage: "Firehose nomad job info changes",
			Action: func(c *cli.Context) error {
				firehose, err := jobs.NewJobListStubFirehose()
				if err != nil {
					log.Fatal(err)
				}

				manager := helper.NewManager(firehose)
				if err := manager.Start(); err != nil {
					log.Fatal(err)
				}

				return nil
			},
		},
		{
			Name:  "deployments",
			Usage: "Firehose nomad deployment changes",
			Action: func(c *cli.Context) error {
				firehose, err := deployments.NewFirehose()
				if err != nil {
					log.Fatal(err)
				}

				manager := helper.NewManager(firehose)
				if err := manager.Start(); err != nil {
					log.Fatal(err)
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

		if c.String("log-format") == "json" {
			log.SetFormatter(&log.JSONFormatter{})
		}

		if c.String("log-format") == "gelf" {
			log.SetFormatter(&gelf.GelfFormatter{})
		}

		return nil
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	app.Run(os.Args)
}
