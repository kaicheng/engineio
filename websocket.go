package engineio

import (
	"github.com/kaicheng/goport/engineio/parser"
	"github.com/kaicheng/goport/events"
)

type websocketTransport struct {
	events.EventEmitter

	socket       *Socket
	wswritable   bool
	wsReadyState string
}

func NewWebsocketTransport(req *Request) Transport {
	return new(websocketTransport)
}

func (trans *websocketTransport) onData(data []byte) {
	pkt := parser.DecodePacket(data)
	trans.onPacket(&pkt)
}

func (trans *websocketTransport) send(packets []*parser.Packet) {
	for _, pkt := range packets {
		parser.EncodePacket(pkt, false, func(data []byte) {
			trans.wswritable = false
			/*
				trans.socket.send(data, func(err) {
					if err {
						trans.onError("write error", err)
					}
					trans.writable = true
					trans.Emit("drain")
				})
			*/
		})
	}
}

func (trans *websocketTransport) doClose(fn func()) {
	trans.socket.Close()
	if fn != nil {
		fn()
	}
}

func (trans *websocketTransport) onClose() {
	trans.wsReadyState = "closed"
	trans.Emit("close")
}

func (trans *websocketTransport) onPacket(pkt *parser.Packet) {
	trans.Emit("packet", pkt)
}

func (trans *websocketTransport) onError(msg, desc string) {
	// err := error{"TransportError", msg, desc}
	// trans.Emit("error", err)
}

func (trans *websocketTransport) close(fn func()) {
	trans.wsReadyState = "closing"
	trans.doClose(fn)
}

func (trans *websocketTransport) onRequest(req *Request) {
	// trans.req = req
}

func (trans *websocketTransport) Name() string {
	return "websocket"
}

func (trans *websocketTransport) readyState() string {
	return trans.wsReadyState
}

func (trans *websocketTransport) setReadyState(state string) {
	trans.wsReadyState = state
}

func (trans *websocketTransport) setSid(sid string) {
}

func (trans *websocketTransport) setMaxHTTPBufferSize(size int) {
}

func (trans *websocketTransport) setSupportsBinary(b bool) {
}

func (trans *websocketTransport) writable() bool {
	return trans.wswritable
}
