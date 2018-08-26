# Logrus GELF Log Formatter
[Logrus](https://github.com/sirupsen/logrus) Only a formatter, outputs the log message in the GELF JSON format as decribed by Graylog 2.

If you are looking for a GELF hook capable of sending directly to graylog, please use this: https://github.com/gemnasium/logrus-graylog-hook

The reason for using just a formatter is that you can write files or directly to stdout in the expected format and then separately ship the
lines to graylog, without stealing CPU time from your go app.

## Installation
To install formatter, use `go get`:

```sh
$ go get github.com/seatgeek/logrus-gelf-formatter
```

## Usage
Here is how it should be used:

```go
package main

import (
	"github.com/sirupsen/logrus"
	gelf "github.com/seatgeek/logrus-gelf-formatter"
)

var log = logrus.New()

func init() {
	log.Formatter = new(gelf.GelfFormatter)
	log.Level = logrus.DebugLevel
}

func main() {
	log.WithFields(logrus.Fields{
		"prefix": "main",
		"animal": "walrus",
		"number": 8,
	}).Debug("Started observing beach")

	log.WithFields(logrus.Fields{
		"prefix":      "sensor",
		"temperature": -4,
	}).Info("Temperature changes")
}
```
