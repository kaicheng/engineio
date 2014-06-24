package engineio

import (
	"github.com/gorilla/websocket"
	"github.com/kaicheng/goport/engineio/parser"
	"net/http"
)

type WebSocket struct {
	TransportBase

	conn    *websocket.Conn
	writeCh chan []byte
	stopCh  chan bool
}

func NewWebSocketTransport(req *Request) Transport {
	ws := new(WebSocket)
	ws.InitWebSocket(req)
	return ws
}

func websocketReadWorker(ws *WebSocket) {
	for {
		select {
		case <-ws.stopCh:
			return
		default:
		}
		_, p, err := ws.conn.ReadMessage()
		debug("websocket received ", string(p))
		if err != nil {
			debug("websocket: read error", err)
			break
		}
		ws.onData(p)
	}
}

func websocketWriteWorker(ws *WebSocket) {
	for {
		select {
		case data := <-ws.writeCh:
			debug("websocket writing ", string(data))
			msgType := websocket.TextMessage
			if data[0] < 20 {
				msgType = websocket.BinaryMessage
			}
			if err := ws.conn.WriteMessage(msgType, data); err != nil {
				debug("websocket: write error", err)
				return
			}
		case <-ws.stopCh:
			return
		}
	}
}

var upgrader websocket.Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {},
}

func (ws *WebSocket) InitWebSocket(req *Request) {
	debug("InitWebSocket")
	ws.initTransportBase(req)
	ws.name = "websocket"

	conn, err := upgrader.Upgrade(req.res, req.httpReq, nil)
	if err != nil {
		debug("InitWebSocket: upgrade fail with err", err)
		ws.transReadyState = "closed"
		return
	}
	ws.conn = conn

	ws.writeCh = make(chan []byte, 1)
	ws.stopCh = make(chan bool, 2)

	go websocketReadWorker(ws)
	go websocketWriteWorker(ws)
}

func (ws *WebSocket) send(pkts []*parser.Packet) {
	for _, pkt := range pkts {
		parser.EncodePacket(pkt, ws.supportsBinary, func(data []byte) {
			ws.writeCh <- data
			ws.Emit("drain")
		})
	}
}

func (ws *WebSocket) tryWritable(fn, def func()) {
	// FIXME: may block if closed.
	fn()
}

func (ws *WebSocket) doClose() {
	debug("websocket closing")
	select {
	case ws.stopCh <- true:
	default:
	}
	select {
	case ws.stopCh <- true:
	default:
	}
	ws.conn.Close()
}
