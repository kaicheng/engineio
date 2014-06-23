package engineio

import (
	"github.com/kaicheng/goport/engineio/parser"
	"github.com/gorilla/websocket"
)

type WebSocket struct {
	TransportBase

	conn         *websocket.Conn
}

func NewWebSocketTransport(req *Request) Transport {
	ws := new(WebSocket)
	ws.InitWebSocket(req)
	return ws
}

func websocketReadWorker(ws *WebSocket) {
	for {
		_, p, err := ws.conn.ReadMessage()
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
			if err := ws.conn.WriteMessage(); err != nil {
				return
			}
		}
	}
}

func (ws *WebSocket) InitWebSocket(req *Request) {
	ws.initTransportBase(req)
	ws.name = "websocket"

	upgrader := websocket.Upgrader{
		ReadBufferSize: 1024,
		WriteBufferSize: 1024,
	}
	ws.conn, err := upgrader.Upgrade(req.res, req.httpReq, nil)
	if err != nil {
		ws.transReadyState = "closed"
		return
	}

	go ws.websocketReadWorker(ws)
	go ws.websocketWriteWorker(ws)

	ws.readyCh <- true	
}

func (ws *WebSocket) send(pkts []*parser.Packet) {
	for _, pkt := range pkts {
		parser.encodePacket(packets[i], this.supportsBinary, func(data []byte) {
			ws.writeCh <- data
			ws.Emit("drain")
		})
	}
	ws.readyCh <- true
}

func (ws *WebSocket) doClose() {
}