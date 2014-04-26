package engineio

import (
	"fmt"
	"time"

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

	writeBuffer []*parser.Packet

	checkIntervalTimer  *time.Timer
	upgradeTimeoutTimer *time.Timer
	pingTimeoutTimer    *time.Timer
}

func newSocket(id string, srv *Server, transport Transport, req *Request) *Socket {
	socket := new(Socket)
	socket.InitEventEmitter()

	socket.id = id
	socket.server = srv
	socket.upgraded = false
	socket.readyState = "opening"
	socket.request = req
	socket.setTransport(transport)

	// TODO: make capacity configurable
	socket.writeBuffer = make([]*parser.Packet, 10)[0:0]

	socket.onOpen()
	return socket
}

func (socket *Socket) onOpen() {
	socket.readyState = "open"
	socket.transport.setSid(socket.id)
	socket.sendPacket("open", []byte(fmt.Sprintf("{\"sid\":\"%s\", \"upgrades\":\"%s\", \"pingInterval\":\"%s\", \"pingTimeout\":\"%s\",}",
		socket.id, socket.getAvailableUpgrades(), socket.server.pingInterval, socket.server.pingTimeout)))

	socket.Emit("open")
	socket.setPingTimeout()
}

func (socket *Socket) onClose(reason, desc string) {
	if "closed" != socket.readyState {
		socket.pingTimeoutTimer.Stop()
		socket.checkIntervalTimer.Stop()
		socket.checkIntervalTimer = nil
		socket.upgradeTimeoutTimer.Stop()

		socket.clearTransport()
		socket.readyState = "closed"
		socket.Emit("close", reason, desc)
		socket.writeBuffer = socket.writeBuffer[0:0]
	}
}

func (socket *Socket) sendPacket(strType string, data []byte) {
	if "closing" != socket.readyState {
		packet := &parser.Packet{Type: strType, Data: data}
		socket.Emit("packetCreate", packet)
		socket.writeBuffer = append(socket.writeBuffer, packet)
		socket.flush()
	}
}

func (socket *Socket) onPacket(packet *parser.Packet) {
	if "open" == socket.readyState {
		socket.Emit("packet", packet)

		socket.setPingTimeout()

		switch packet.Type {
		case "ping":
			socket.sendPacket("pong", nil)
			socket.Emit("heartbeat")
		case "error":
			socket.onClose("parse error", "")
		case "message":
			socket.Emit("data", packet.Data)
			socket.Emit("message", packet.Data)
		}
	}
}

func (socket *Socket) onError(err string) {
	socket.onClose("transport error", err)
}

func (socket *Socket) setPingTimeout() {
	if socket.pingTimeoutTimer != nil {
		socket.pingTimeoutTimer.Stop()
	}
	socket.pingTimeoutTimer = time.AfterFunc(socket.server.pingInterval+socket.server.pingTimeout, func() {
		socket.onClose("ping timeout", "")
	})
}

func (socket *Socket) clearTransport() {
	socket.checkIntervalTimer.Stop()
	socket.pingTimeoutTimer.Stop()
}

func (socket *Socket) setupSendCallback() {
}

func (socket *Socket) send(data []byte) {
	socket.sendPacket("message", data)
}

func (socket *Socket) write(data []byte) {
	socket.send(data)
}

func (socket *Socket) flush() {
	if "closed" != socket.readyState && socket.transport.writable() && len(socket.writeBuffer) > 0 {
		socket.Emit("flush", socket.writeBuffer)
		socket.server.Emit("flush", socket.writeBuffer)
		buf := socket.writeBuffer
		socket.writeBuffer = make([]*parser.Packet, 10)[0:0]
		socket.transport.send(buf)
		socket.Emit("drain")
		socket.server.Emit("drain", socket)
	}
}

func (socket *Socket) getAvailableUpgrades() []string {
	return []string{}
}

func (socket *Socket) setTransport(transport Transport) {
	socket.transport = transport
	transport.Once("error", socket.onError)
	transport.On("packet", socket.onPacket)
	transport.On("drain", socket.flush)
	transport.Once("close", func() { socket.onClose("transport close", "") })
	socket.setupSendCallback()
}

func (socket *Socket) close() {
	if "open" == socket.readyState {
		socket.readyState = "closing"
		socket.transport.close(func() {
			socket.onClose("froced close", "")
		})
	}
}
