package sink

// Sink ...
type Sink interface {
	Start() error
	Stop()
	Put(data []byte) error
}
