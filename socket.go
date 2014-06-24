package engineio

import (
	"encoding/json"
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
	Request    *Request
	Transport  Transport

	WriteBuffer []*parser.Packet

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
	socket.Request = req
	socket.setTransport(transport)

	// TODO: make capacity configurable
	socket.WriteBuffer = make([]*parser.Packet, 10)[0:0]

	socket.onOpen()
	return socket
}

func (socket *Socket) onOpen() {
	socket.readyState = "open"
	socket.Transport.setSid(socket.id)
	pingInterval := (int64)(socket.server.pingInterval / time.Millisecond)
	pingTimeout := (int64)(socket.server.pingTimeout / time.Millisecond)
	upgrades, _ := json.Marshal(socket.getAvailableUpgrades())
	socket.sendPacket("open", []byte(fmt.Sprintf("{\"sid\":\"%s\",\"upgrades\":%s,\"pingInterval\":%d, \"pingTimeout\":%d}",
		socket.id, upgrades, pingInterval, pingTimeout)))

	socket.Emit("open")
	socket.setPingTimeout()
}

func (socket *Socket) onClose(reason, desc string) {
	if "closed" != socket.readyState {
		socket.pingTimeoutTimer.Stop()
		if socket.checkIntervalTimer != nil {
			socket.checkIntervalTimer.Stop()
		}
		socket.checkIntervalTimer = nil
		if socket.upgradeTimeoutTimer != nil {
			socket.upgradeTimeoutTimer.Stop()
		}

		socket.clearTransport()
		socket.readyState = "closed"
		socket.Emit("close", reason, desc)
		socket.WriteBuffer = socket.WriteBuffer[0:0]
	}
}

func (socket *Socket) sendPacket(strType string, data []byte) {
	if "closing" != socket.readyState {
		debug(fmt.Sprintf("sending packet \"%s\" (\"%s\")", strType, string(data)))
		packet := &parser.Packet{Type: strType, Data: data}
		socket.Emit("packetCreate", packet)
		socket.WriteBuffer = append(socket.WriteBuffer, packet)
		socket.flush()
	}
}

func (socket *Socket) sendBinPacket(strType string, data []byte) {
	if "closing" != socket.readyState {
		debug(fmt.Sprintf("sending packet \"%s\" (\"%s\")", strType, string(data)))
		packet := &parser.Packet{Type: strType, Data: data, IsBin: true}
		socket.Emit("packetCreate", packet)
		socket.WriteBuffer = append(socket.WriteBuffer, packet)
		socket.flush()
	}
}

func (socket *Socket) onPacket(packet *parser.Packet) {
	if "open" == socket.readyState {
		debug("packet ", packet.Type)
		debug("packet.Data", string(packet.Data))
		socket.Emit("packet", packet)

		socket.setPingTimeout()

		switch packet.Type {
		case "ping":
			debug("got ping")
			socket.sendPacket("pong", nil)
			socket.Emit("heartbeat")
		case "error":
			socket.onClose("parse error", "")
		case "message":
			socket.Emit("data", packet.Data)
			socket.Emit("message", packet.Data)
		}
	} else {
		debug("packet received with closed socket")
	}
}

func (socket *Socket) OnError(err string) {
	debug("transport error")
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
	socket.Transport.On("error", func(arg interface{}) {
		debug("error triggered by discarded transport")
	})
	socket.pingTimeoutTimer.Stop()
}

func (socket *Socket) setupSendCallback() {
}

func (socket *Socket) Send(data []byte) {
	socket.sendPacket("message", data)
}

func (socket *Socket) SendBin(data []byte) {
	socket.sendBinPacket("message", data)
}

func (socket *Socket) Write(data []byte) {
	socket.Send(data)
}

func (socket *Socket) flush() {
	if "closed" != socket.readyState && len(socket.WriteBuffer) > 0 {
		socket.Transport.tryWritable(func() {
		    debug("flusing buffer to transport")
			socket.Emit("flush", socket.WriteBuffer)
			socket.server.Emit("flush", socket.WriteBuffer)
			buf := socket.WriteBuffer
			socket.WriteBuffer = make([]*parser.Packet, 10)[0:0]
			socket.Transport.send(buf)
			socket.Emit("drain")
			socket.server.Emit("drain", socket)
		}, nil)
	}
}

func (socket *Socket) getAvailableUpgrades() []string {
	return socket.server.upgrades(socket.Transport.Name())
}

func (socket *Socket) setTransport(transport Transport) {
	socket.Transport = transport
	transport.Once("error", socket.OnError)
	transport.On("packet", socket.onPacket)
	transport.On("drain", socket.flush)
	transport.Once("close", func() { socket.onClose("transport close", "") })
	socket.setupSendCallback()
}

type funcBag struct {
	fn func(*parser.Packet)
}

func (socket *Socket) maybeUpgrade(transport Transport) {
	debug(fmt.Sprintf("might upgrade socket transport from \"%s\" to \"%s\"",
		socket.Transport.Name(), transport.Name()))

	onPacket := new(funcBag)
	onPacket.fn = func(pkt *parser.Packet) {
		if "ping" == pkt.Type && "probe" == string(pkt.Data) {
			transport.send([]*parser.Packet{&parser.Packet{Type: "pong", Data: []byte("probe")}})
		} else if "upgrade" == pkt.Type && socket.readyState == "open" {
			debug("got upgrade packet - upgrading")
			socket.upgraded = true
			socket.clearTransport()
			socket.setTransport(transport)
			socket.Emit("upgrade", transport)
			socket.flush()
			transport.RemoveListener("packet", onPacket.fn)
			debug(fmt.Sprintf("upgrade to \"%s\" finishes", transport.Name()))
		}
	}

	transport.On("packet", onPacket.fn)
}

func (socket *Socket) Close() {
	if "open" == socket.readyState {
		socket.readyState = "closing"
		socket.Transport.close(func() {
			socket.onClose("froced close", "")
		})
	}
}
