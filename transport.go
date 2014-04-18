package trans

import (
    "github.com/kaicheng/goport/events"
)

type Transport interface {
    events.Emitter

    setReadyState(string)
    readState() string

    onRequest(*Request)
    close()
    onError(string, error)
    onPacket(*Packet)
    onData(string)
    onClose()
}

type transportCreator func(*Request)*Transport

transports := map[string]transportCreator {
    "websocket" : NewWebsocketTransport,
}
