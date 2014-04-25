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
	setMaxHTTPBufferSize(size int)
	setSupportsBinary(b bool)
	writable() bool
}

type transportCreator func(*Request) Transport

var transports = map[string]transportCreator{
	"websocket": NewWebsocketTransport,
	"polling":   NewPollingTransport,
}

var noopPkt = parser.Packet{Type: "noop"}

type TransportBase struct {
	events.EventEmitter

	doClose         func(func())
	transReadyState string
	req             *Request
	name            string
	sid             string
	supportsBinary  bool
}

func (trans *TransportBase) initTransportBase(req *Request) {
	trans.transReadyState = "opening"
	trans.doClose = func(func()) {}
}

func (trans *TransportBase) setReadyState(state string) {
	trans.transReadyState = state
}

func (trans *TransportBase) readyState() string {
	return trans.transReadyState
}

func (trans *TransportBase) onRequest(req *Request) {
	trans.req = req
}

func (trans *TransportBase) close(fn func()) {
	trans.transReadyState = "closing"
	if fn == nil {
		fn = func() {}
	}
	trans.doClose(fn)
}

func (trans *TransportBase) onError(msg, desc string) {
	trans.Emit("error", &Error{Msg: msg, Type: "TransportError", Desc: desc})
}

func (trans *TransportBase) onPacket(pkt *parser.Packet) {
	trans.Emit("packet", pkt)
}

func (trans *TransportBase) onData(data []byte) {
	pkt := parser.DecodePacket(data)
	trans.onPacket(&pkt)
}

func (trans *TransportBase) onClose() {
	trans.transReadyState = "closed"
	trans.Emit("close")
}

func (trans *TransportBase) Name() string {
	return trans.name
}

func (trans *TransportBase) setSid(sid string) {
	trans.sid = sid
}

func (trans *TransportBase) setMaxHTTPBufferSize(size int) {}

func (trans *TransportBase) setSupportsBinary(b bool) {
	trans.supportsBinary = b
}
