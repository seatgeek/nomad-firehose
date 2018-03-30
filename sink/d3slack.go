package sink

import (
	"os"
	"time"
  "net/http"
	"fmt"
	"bytes"
  "encoding/json"

	log "github.com/sirupsen/logrus"
)

type SlackPayload struct {
	Parse       string       `json:"parse,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconUrl     string       `json:"icon_url,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Channel     string       `json:"channel"`
	Text        string       `json:"text,omitempty"`
	LinkNames   string       `json:"link_names,omitempty"`
	//Attachments []Attachment `json:"attachments,omitempty"`
}

func newSlackPayload(name string, msg string) SlackPayload {
	return SlackPayload{
		Username:   name,
		Text:       msg,
    IconEmoji:  ":doge:",
	}
}

// D3SlackSink ...
type D3SlackSink struct {
	webhook    string
  bot_name   string
	stopCh     chan interface{}
	putCh      chan []byte
}

// New ...
func NewD3Slack() (*D3SlackSink, error) {
	wh := os.Getenv("D3_SLACK_WEBHOOK")
	if wh == "" {
		return nil, fmt.Errorf("[sink/d3slack] Missing D3_SLACK_WEBHOOK (example: https://hooks.slack.com/services/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)")
	}
  bn := os.Getenv("D3_BOT_NAME")
	if wh == "" {
		return nil, fmt.Errorf("[sink/d3slack] Missing D3_BOT_NAME")
	}

	return &D3SlackSink{
		webhook:   wh,
    bot_name:  bn,
		stopCh: make(chan interface{}),
		putCh:  make(chan []byte, 1000),
	}, nil
}

// Start ...
func (s *D3SlackSink) Start() error {
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
func (s *D3SlackSink) Stop() {
	log.Debugf("[sink/d3slack] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Debugf("[sink/d3slack] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
}

// Put ..
func (s *D3SlackSink) Put(data []byte) error {
	s.putCh <- data
	return nil
}

func (s *D3SlackSink) write() {
	log.Infof("[sink/d3slack] Starting ...")

	for {
		select {
		case data := <-s.putCh:
			//conn := s.pool.Get()
			//if _, err := conn.Do("RPUSH", s.key, data); err != nil {
				//log.Infof("[sink/d3slack] %s", err)
			//} else {
				//log.Infof("[sink/d3slack] Published to key '%s'", s.key)
			//}
			//conn.Close()
      var r []byte
	    //msg := fmt.Sprintf("%s: Notable event for job `%s` (AllocationID: `%s`) in `owf-dev` at: *%s/%s/%s*", t_str, update.JobID, update.AllocationID, update.TaskEvent.Type, update.TaskEvent.Message, update.TaskEvent.DisplayMessage)
	    slack := newSlackPayload(s.bot_name, string(data))
	    slack.Channel = "special_ed"
      r, _ = json.Marshal(slack)

      req, err := http.NewRequest("POST", s.webhook, bytes.NewBuffer(r))
      if err != nil {
	      fmt.Sprintf("Error creating request: %s", err.Error())
		    return
      }
      req.Header.Set("Content-Type", "application/json")

      client := &http.Client{
	      Timeout: 5 * time.Second,
      }
      resp, err := client.Do(req)
      if err != nil {
	      fmt.Sprintf("Error sending request: %s", err.Error())
		    return
      }
      defer resp.Body.Close()
		}
	}
}
