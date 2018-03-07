package main

import (
	"os"
	"sort"

	"app/command/allocations"
	"app/command/deployments"
	"app/command/evaluations"
	"app/command/jobs"
	"app/command/nodes"

	gelf "github.com/seatgeek/logrus-gelf-formatter"
	log "github.com/sirupsen/logrus"
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
		{
			Name:  "deployments",
			Usage: "Firehose nomad deployment changes",
			Action: func(c *cli.Context) error {
				firehose, err := deployments.NewFirehose()
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
