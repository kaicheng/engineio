package engineio

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kaicheng/engineio/parser"
	"github.com/kaicheng/events"
)

type Socket struct {
	events.EventEmitter

	id         string
	server     *Server
	upgraded   bool
	readyState string
	Request    *Request
	Transport  Transport

	writeBuffer []*parser.Packet

	checkIntervalTimer  *ticker
	upgradeTimeoutTimer *time.Timer
	pingTimeoutTimer    *time.Timer

	bufferLock     sync.Mutex
	timerLock      sync.Mutex
	readyStateLock sync.Mutex
	transportLock  sync.Mutex
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
	socket.writeBuffer = make([]*parser.Packet, 10)[0:0]

	socket.onOpen()
	return socket
}

func (socket *Socket) WriteBuffer() []*parser.Packet {
	socket.bufferLock.Lock()
	defer socket.bufferLock.Unlock()
	return socket.writeBuffer
}

func (socket *Socket) SetWriteBuffer(buf []*parser.Packet) {
	socket.bufferLock.Lock()
	defer socket.bufferLock.Unlock()
	socket.writeBuffer = buf
}

func (socket *Socket) getTransport() Transport {
	socket.transportLock.Lock()
	defer socket.transportLock.Unlock()
	return socket.Transport
}

func (socket *Socket) onOpen() {
	socket.readyState = "open"
	socket.getTransport().setSid(socket.id)
	pingInterval := (int64)(socket.server.pingInterval / time.Millisecond)
	pingTimeout := (int64)(socket.server.pingTimeout / time.Millisecond)
	upgrades, _ := json.Marshal(socket.getAvailableUpgrades())
	socket.sendPacket("open", []byte(fmt.Sprintf("{\"sid\":\"%s\",\"upgrades\":%s,\"pingInterval\":%d, \"pingTimeout\":%d}",
		socket.id, upgrades, pingInterval, pingTimeout)))

	socket.Emit("open")
	socket.setPingTimeout()
}

func (socket *Socket) onClose(reason, desc string) {
	socket.readyStateLock.Lock()
	if "closed" != socket.readyState {
		socket.timerLock.Lock()
		if socket.pingTimeoutTimer != nil {
			socket.pingTimeoutTimer.Stop()
		}
		socket.pingTimeoutTimer = nil
		if socket.checkIntervalTimer != nil {
			socket.checkIntervalTimer.stop()
		}
		socket.checkIntervalTimer = nil
		if socket.upgradeTimeoutTimer != nil {
			socket.upgradeTimeoutTimer.Stop()
		}
		socket.upgradeTimeoutTimer = nil
		socket.timerLock.Unlock()
		socket.clearTransport()
		socket.readyState = "closed"
		socket.readyStateLock.Unlock()
		socket.Emit("close", reason, desc)
		socket.bufferLock.Lock()
		socket.writeBuffer = socket.writeBuffer[0:0]
		socket.bufferLock.Unlock()
	} else {
		socket.readyStateLock.Unlock()
	}
}

func (socket *Socket) sendPacket(strType string, data []byte) {
	socket.readyStateLock.Lock()
	state := socket.readyState
	socket.readyStateLock.Unlock()
	if "closing" != state {
		debug(fmt.Sprintf("sending packet \"%s\" (\"%s\")", strType, string(data)))
		packet := &parser.Packet{Type: strType, Data: data}
		socket.Emit("packetCreate", packet)
		socket.bufferLock.Lock()
		socket.writeBuffer = append(socket.writeBuffer, packet)
		socket.bufferLock.Unlock()
		socket.flush()
	}
}

func (socket *Socket) sendBinPacket(strType string, data []byte) {
	socket.readyStateLock.Lock()
	state := socket.readyState
	socket.readyStateLock.Unlock()
	if "closing" != state {
		debug(fmt.Sprintf("sending packet \"%s\" (\"%s\")", strType, string(data)))
		packet := &parser.Packet{Type: strType, Data: data, IsBin: true}
		socket.Emit("packetCreate", packet)
		socket.bufferLock.Lock()
		socket.writeBuffer = append(socket.writeBuffer, packet)
		socket.bufferLock.Unlock()
		socket.flush()
	}
}

func (socket *Socket) onPacket(packet *parser.Packet) {
	socket.readyStateLock.Lock()
	state := socket.readyState
	socket.readyStateLock.Unlock()

	if "open" == state {
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
	socket.timerLock.Lock()
	defer socket.timerLock.Unlock()
	if socket.pingTimeoutTimer != nil {
		socket.pingTimeoutTimer.Stop()
	}
	socket.pingTimeoutTimer = time.AfterFunc(socket.server.pingInterval+socket.server.pingTimeout, func() {
		socket.onClose("ping timeout", "")
	})
}

func (socket *Socket) clearTransport() {
	socket.getTransport().On("error", func(arg interface{}) {
		debug("error triggered by discarded transport")
	})
	socket.timerLock.Lock()
	defer socket.timerLock.Unlock()
	if socket.pingTimeoutTimer != nil {
		socket.pingTimeoutTimer.Stop()
	}
	socket.pingTimeoutTimer = nil
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
	socket.readyStateLock.Lock()
	state := socket.readyState
	socket.readyStateLock.Unlock()
	socket.bufferLock.Lock()
	if "closed" != state && len(socket.writeBuffer) > 0 {
		socket.bufferLock.Unlock()
		trans := socket.getTransport()
		trans.tryWritable(func() {
			debug("flusing buffer to transport")
			socket.bufferLock.Lock()
			buf := socket.writeBuffer
			socket.writeBuffer = make([]*parser.Packet, 10)[0:0]
			socket.bufferLock.Unlock()
			socket.Emit("flush", buf)
			socket.server.Emit("flush", buf)
			trans.send(buf)
			socket.Emit("drain")
			socket.server.Emit("drain", socket)
		}, nil)
	} else {
		socket.bufferLock.Unlock()
	}
}

func (socket *Socket) getAvailableUpgrades() []string {
	return socket.server.upgrades(socket.Transport.Name())
}

func (socket *Socket) setTransport(transport Transport) {
	socket.transportLock.Lock()
	socket.Transport = transport
	socket.transportLock.Unlock()
	transport.Once("error", socket.OnError)
	transport.On("packet", socket.onPacket)
	transport.On("drain", socket.flush)
	transport.Once("close", func() {
		debug("transport on close, closing")
		socket.onClose("transport close", "")
	})
	socket.setupSendCallback()
}

type funcBag struct {
	fn func(*parser.Packet)
}

func (socket *Socket) maybeUpgrade(transport Transport) {
	debug(fmt.Sprintf("might upgrade socket transport from \"%s\" to \"%s\"",
		socket.getTransport().Name(), transport.Name()))

	socket.timerLock.Lock()
	socket.upgradeTimeoutTimer = time.AfterFunc(socket.server.upgradeTimeout,
		func() {
			debug("client did not complete upgrade - closing tansport")
			if "open" == transport.readyState() {
				transport.close(nil)
			}
		})
	socket.timerLock.Unlock()

	onPacket := new(funcBag)
	onPacket.fn = func(pkt *parser.Packet) {
		if "ping" == pkt.Type && "probe" == string(pkt.Data) {
			transport.send([]*parser.Packet{&parser.Packet{Type: "pong", Data: []byte("probe")}})
			socket.timerLock.Lock()
			if socket.checkIntervalTimer != nil {
				socket.checkIntervalTimer.stop()
			}
			// TODO: set as a parameter
			socket.checkIntervalTimer = newTicker(100 * time.Millisecond)
			go func(c <-chan time.Time, end <-chan bool) {
				for {
					select {
					case <-c:
						trans := socket.getTransport()
						if "polling" == trans.Name() {
							trans.tryWritable(func() {
								debug("writing a noop packet to polling for fast upgrade")
								trans.send([]*parser.Packet{&parser.Packet{Type: "noop"}})
							}, nil)
						}
					case <-end:
						return
					}
				}
			}(socket.checkIntervalTimer.c, socket.checkIntervalTimer.end)
			socket.timerLock.Unlock()
		} else if "upgrade" == pkt.Type {
			socket.readyStateLock.Lock()
			state := socket.readyState
			socket.readyStateLock.Unlock()
			if state == "open" {
				debug("got upgrade packet - upgrading")
				socket.timerLock.Lock()
				socket.upgradeTimeoutTimer.Stop()
				socket.timerLock.Unlock()
				transport.RemoveListener("packet", onPacket.fn)
				socket.upgraded = true
				socket.clearTransport()
				socket.setTransport(transport)
				socket.Emit("upgrade", transport)
				socket.flush()
				socket.timerLock.Lock()
				socket.checkIntervalTimer.stop()
				socket.checkIntervalTimer = nil
				socket.timerLock.Unlock()
				debug(fmt.Sprintf("upgrade to \"%s\" finishes", transport.Name()))
			}
		} else {
			debug("invalid packet during upgrade")
			transport.close(nil)
		}
	}

	transport.On("packet", onPacket.fn)
}

func (socket *Socket) Close() {
	socket.readyStateLock.Lock()
	if "open" == socket.readyState {
		socket.readyState = "closing"
		socket.readyStateLock.Unlock()
		socket.getTransport().close(func() {
			socket.onClose("forced close", "")
		})
	} else {
		socket.readyStateLock.Unlock()
	}
}
