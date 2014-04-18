package engineio

import (
	"github.com/kaicheng/goport/events"
)

type Socket struct {
    events.EventEmitter

    id string
    server *Server
    upgraded bool
    readyState string
    request *Request
    transport Transport
}

func (socket *Socket) onOpen() {
    socket.readyState = "open"
    this.transport.sid = socket.id
    socket.sendPacket("open")

    socket.Emit("open")
    socket.setPingTimeout()
}

func (socket *Socket) onPacket(packet) {
    if ("open" == socket.readyState) {
        socket.Emit("packet", packet)

        socket.setPingTimeout()

        switch packet.type {
            case "ping":
                socket.sendPacket("pong")
            case "error":
                socket.onClose("parse error")
            case "message"
                socket.Emit("data", packet.data)
                socket.Emit("message", packet.data)
        }
    }
}

func (socket *Socket) setPingTimeout() {
}

func (socket *Socket) setTransport(transport *Transport) {
}

func (socket *Socket) Close() {
    if ("open" == this.readyState) {
        socket.readyState = "closing"
        socket.transport.Close(func () {
            socket.onClose("froced close")
        })
    }
}
