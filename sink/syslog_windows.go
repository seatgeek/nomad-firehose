// +build windows

package sink

import (
	"fmt"
)

type SyslogSink struct{}

// NewSyslog ...
func NewSyslog() (*SyslogSink, error) {
	return &SyslogSink{}, fmt.Errorf("[sink/syslog] ERROR - not supported on Windows :(")
}

// Put ...
func (s *SyslogSink) Put(_ []byte) error {
	return nil
}

// Start ...
func (s *SyslogSink) Start() error {
	return nil
}

// Stop ...
func (s *SyslogSink) Stop() {
	return
}
