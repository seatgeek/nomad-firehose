package sink

// Sink ...
type Sink interface {
	Start() error
	Stop()
	Put(key string, data []byte) error
}

type Data struct {
	key   string
	value []byte
}
