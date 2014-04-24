package engineio

import (
	"github.com/kaicheng/goport/engineio/parser"
	"github.com/kaicheng/goport/events"
)

type Transport interface {
	events.EventEmitterInt

	setReadyState(string)
	readyState() string

	onRequest(*Request)
	close(func())
	onError(string, string)
	onPacket(*parser.Packet)
	onData([]byte)
	onClose()

	Name() string
	setSid(sid string)
}

type transportCreator func(*Request) Transport

var transports = map[string]transportCreator{
	"websocket": NewWebsocketTransport,
}
