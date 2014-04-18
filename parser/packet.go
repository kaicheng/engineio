package parser

const (
	Open = iota
	Close
	Ping
	Pong
	Message
	Upgrade
	Noop
)

type Package struct {
	Type int
	Data string
}
