package engineio

import (
	"github.com/kaicheng/goport/engineio/parser"
	"github.com/gorilla/websocket"
	"net/http"
)

type WebSocket struct {
	TransportBase

	conn         *websocket.Conn
	readyCh      chan bool
	writeCh      chan []byte
}

func NewWebSocketTransport(req *Request) Transport {
	ws := new(WebSocket)
	ws.InitWebSocket(req)
	return ws
}

func websocketReadWorker(ws *WebSocket) {
	for {
		_, p, err := ws.conn.ReadMessage()
		debug("received ", string(p))
		if err != nil {
			break
		}
		ws.onData(p)
	}
}

func websocketWriteWorker(ws *WebSocket) {
	for {
		select {
		case data := <-ws.writeCh:
			debug("writing ", string(data))
			if err := ws.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}

func (ws *WebSocket) InitWebSocket(req *Request) {
	debug("InitWebSocket")
	ws.initTransportBase(req)
	ws.name = "websocket"

	upgrader := websocket.Upgrader{
		ReadBufferSize: 1024,
		WriteBufferSize: 1024,
		CheckOrigin: func (r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(req.res, req.httpReq, nil)
	if err != nil {
		debug("InitWebSocket: upgrade fail with err", err)
		ws.transReadyState = "closed"
		return
	}
	ws.conn = conn

	ws.readyCh = make(chan bool, 1)
	ws.writeCh = make(chan []byte, 1)

	go websocketReadWorker(ws)
	go websocketWriteWorker(ws)

	ws.readyCh <- true	
}

func (ws *WebSocket) send(pkts []*parser.Packet) {
	for _, pkt := range pkts {
		parser.EncodePacket(pkt, ws.supportsBinary, func(data []byte) {
			ws.writeCh <- data
			ws.Emit("drain")
		})
	}
	ws.readyCh <- true
}

func (ws *WebSocket) tryWritable(fn, def func()) {
	select {
	case <-ws.readyCh:
		fn()
	default:
		if def != nil {
			def()
		}
	}
}

func (ws *WebSocket) doClose() {
}