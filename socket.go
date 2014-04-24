package engineio

import (
	"github.com/kaicheng/goport/engineio/parser"
	"github.com/kaicheng/goport/events"
)

type Socket struct {
	events.EventEmitter

	id         string
	server     *Server
	upgraded   bool
	readyState string
	request    *Request
	transport  Transport
}

func newSocket(id string, srv *Server, transport Transport, req *Request) *Socket {
	socket := new(Socket)
	socket.InitEventEmitter()

	socket.id = id
	socket.server = srv
	socket.upgraded = false
	socket.readyState = "opening"
	socket.request = req
	socket.transport = transport

	return socket
}

func (socket *Socket) onOpen() {
	socket.readyState = "open"
	socket.transport.setSid(socket.id)
	socket.sendPacket("open")

	socket.Emit("open")
	socket.setPingTimeout()
}

func (socket *Socket) onClose(msg string) {

}

func (socket *Socket) sendPacket(pkt string) {

}

func (socket *Socket) onPacket(packet *parser.Packet) {
	if "open" == socket.readyState {
		socket.Emit("packet", packet)

		socket.setPingTimeout()

		switch packet.Type {
		case "ping":
			socket.sendPacket("pong")
		case "error":
			socket.onClose("parse error")
		case "message":
			socket.Emit("data", packet.Data)
			socket.Emit("message", packet.Data)
		}
	}
}

func (socket *Socket) setPingTimeout() {
}

func (socket *Socket) setTransport(transport Transport) {
}

func (socket *Socket) close() {
	if "open" == socket.readyState {
		socket.readyState = "closing"
		socket.transport.close(func() {
			socket.onClose("froced close")
		})
	}
}
