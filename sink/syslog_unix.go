// +build darwin linux

package sink

import (
	"fmt"
	"log/syslog"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

type SyslogSink struct {
	addr     string
	proto    string
	priority syslog.Priority
	tag      string
	stopCh   chan interface{}
	putCh    chan []byte
}

// NewSyslog ...
func NewSyslog() (*SyslogSink, error) {
	syslogProto := os.Getenv("SINK_SYSLOG_PROTO")
	// The log/syslog package has some interesting internal behaviours. If we
	// *do not* supply the network protocol, syslog.Dial assumes we are
	// connecting to a local syslog socket and configures itself for "local"
	// mode, which does not include the local hostname in messages written to
	// the socket (and avoids breaking a standard).
	//
	// See: https://github.com/golang/go/commit/87a6d75012986fb8867b746afcd42f742c119945
	if syslogProto == "" {
		log.Info("[sink/syslog] SINK_SYSLOG_PROTO not set - ignoring this and SINK_SYSLOG_ADDR, as syslog package will default to unixgram and an autodiscovered socket")
	}
	syslogAddr := os.Getenv("SINK_SYSLOG_ADDR")
	if syslogAddr == "" && syslogProto != "" {
		return nil, fmt.Errorf("[sink/syslog] Missing SINK_SYSLOG_ADDR (examples: 192.168.1.100:514")
	}
	syslogTag := os.Getenv("SINK_SYSLOG_TAG")
	if syslogTag == "" {
		log.Info("[sink/syslog] Missing SINK_SYSLOG_TAG - setting to default 'nomad-firehose'")
		syslogTag = "nomad-firehose"
	}

	return &SyslogSink{
		addr:     syslogAddr,
		proto:    syslogProto,
		tag:      syslogTag,
		priority: syslog.LOG_INFO,

		stopCh: make(chan interface{}),
		putCh:  make(chan []byte, 1000),
	}, nil
}

// Start ...
func (s *SyslogSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

	go s.write()

	// wait forever for a stop signal to happen
	for {
		select {
		case <-s.stopCh:
			break
		}
		break
	}

	return nil
}

// Stop ...
func (s *SyslogSink) Stop() {
	log.Infof("[sink/syslog] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Infof("[sink/syslog] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
}

// Put ..
func (s *SyslogSink) Put(data []byte) error {
	s.putCh <- data
	return nil
}

func (s *SyslogSink) write() error {
	log.Infof("[sink/syslog] Starting writer - %s://%s - tag: %s", s.proto, s.addr, s.tag)
	writer, err := syslog.Dial(s.proto, s.addr, syslog.LOG_NOTICE, s.tag)
	if err != nil {
		log.Infof("[sink/syslog] ERROR initializing syslog writer: %q", err)
		return err
	}

	for {
		select {
		case data := <-s.putCh:
			// fmt.Fprint(writer, string(data))
			_, err := writer.Write(data)
			if err != nil {
				log.Infof("[sink/syslog] ERROR writing to syslog: %q", err)
			}
		}
	}
}
