package engineio

import (
	"bytes"
	"github.com/kaicheng/goport/engineio/parser"
	"net/http"
)

type Polling struct {
	TransportBase

	res               http.ResponseWriter
	dataRes           http.ResponseWriter
	dataReq           *Request
	pWritable         bool
	cleanup           func()
	maxHTTPBufferSize int
	shouldClose       func()
	headers           func(req *Request)
	doWrite           func(data []byte)
}

func NewPollingTransport(req *Request) Transport {
	xhr := new(XHR)
	xhr.InitXHR(req)
	return xhr
}

func (poll *Polling) InitPolling(req *Request) {
	poll.initTransportBase(req)
	poll.name = "polling"
	poll.doClose = func(fn func()) {
		if poll.dataReq != nil {
			poll.dataReq.abort()
		}
		if poll.pWritable {
			poll.send([]*parser.Packet{&parser.Packet{Type: "close"}})
			fn()
		} else {
			poll.shouldClose = fn
		}
	}
}

func (poll *Polling) onRequest(req *Request) {
	res := req.res
	switch req.httpReq.Method {
	case "GET":
		poll.onPollRequest(req)
	case "POST":
		poll.onDataRequest(req)
	default:
		res.WriteHeader(500)
		res.Write(nil)
	}
}

func (poll *Polling) onPollRequest(req *Request) {
	res := req.res

	if poll.req != nil {
		poll.onError("overlap from client", "")
		res.WriteHeader(500)
		res.Write(nil)
		return
	}

	poll.req = req
	poll.res = res

	onClose := func() { poll.onError("poll connection closed prematurely", "") }

	// FIXME:
	// req.cleanup = func() {
	//	   req.RemoveListener("close", onClose)
	//     poll.req = nil
	//     poll.res = nil
	// }
	req.On("close", onClose)

	poll.pWritable = true
	poll.Emit("drain")

	if poll.pWritable && poll.shouldClose != nil {
		poll.send([]*parser.Packet{&noopPkt})
	}
}

type funcBag struct {
	onClose func()
	onData  func(data []byte)
	onEnd   func()
	cleanup func()
}

func (poll *Polling) onDataRequest(req *Request) {
	res := req.res

	if poll.dataReq != nil {
		poll.onError("data request overlap from client", "")
		res.WriteHeader(500)
		res.Write(nil)
		return
	}

	bag := funcBag{}

	chunks := new(bytes.Buffer)
	bag.onData = func(data []byte) {
		if len(data)+chunks.Len() > poll.maxHTTPBufferSize {
			chunks.Reset()
		} else {
			chunks.Write(data)
			req.httpReq.Body.Close()
		}
	}

	bag.onClose = func() {
		bag.cleanup()
		poll.onError("data request connection closed prematurely", "")
	}

	bag.onEnd = func() {
		poll.onData(chunks.Next(chunks.Len()))
		res.Header().Set("Content-Length", "2")
		res.Header().Set("Content-Type", "text/html")
		poll.headers(req)
		res.WriteHeader(200)
		res.Write([]byte("ok"))
		bag.cleanup()
	}

	bag.cleanup = func() {
		req.RemoveListener("data", bag.onData)
		req.RemoveListener("end", bag.onEnd)
		req.RemoveListener("close", bag.onClose)
		poll.dataReq = nil
		poll.dataRes = nil
	}

	req.abort = bag.cleanup
	req.On("close", bag.onClose)
	req.On("data", bag.onData)
	req.On("end", bag.onEnd)

	/*
		isBinary := req.query.Get("content-type") == "application/octet/stream"
		if !isBinary {
			req.setEncoding("utf8")
		}
	*/
}

func (poll *Polling) onData(data []byte) {
	parser.DecodePayload(data, func(pkt parser.Packet, index, total int) {
		if pkt.Type == "close" {
			poll.onClose()
			return
		}
		poll.onPacket(&pkt)
	})
}

func (poll *Polling) send(pkts []*parser.Packet) {
	if poll.shouldClose != nil {
		pkts = append(pkts, &parser.Packet{Type: "close"})
		poll.shouldClose()
		poll.shouldClose = nil
	}

	parser.EncodePayload(pkts, poll.supportsBinary, func(data []byte) {
		poll.write(data)
	})
}

func (poll *Polling) write(data []byte) {
	poll.doWrite(data)
	poll.req.cleanup()
	poll.pWritable = false
}

func (poll *Polling) setMaxHTTPBufferSize(size int) {
	poll.maxHTTPBufferSize = size
}

func (poll *Polling) writable() bool {
	return poll.pWritable
}
