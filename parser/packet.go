package parser

const (
	Open byte = iota
	Close
	Ping
	Pong
	Message
	Upgrade
	Noop
)

var Packets = map[string]byte{
	"open":    Open,
	"close":   Close,
	"ping":    Ping,
	"pong":    Pong,
	"message": Message,
	"upgrade": Upgrade,
	"noop":    Noop,
}

var PacketsList = []string{
	"open",
	"close",
	"ping",
	"pong",
	"message",
	"upgrade",
	"noop",
}

type Packet struct {
	Type string
	Data []byte
}
