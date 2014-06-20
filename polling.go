package engineio

import (
	"bytes"
	"fmt"
	"github.com/kaicheng/goport/engineio/parser"
	"sync/atomic"
)

type Polling struct {
	TransportBase

	cleanup           func()
	maxHTTPBufferSize int
	shouldClose       func()
	headers           func(req *Request)
	doWrite           func(req *Request, data []byte)

	reqGuard  int32
	dataGuard int32

	writeCh chan []byte
	readyCh chan bool
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
		poll.tryWritable(
			func() {
				poll.send([]*parser.Packet{&parser.Packet{Type: "close"}})
				fn()
			},
			func() {
				poll.shouldClose = fn
			})
	}
}

func (poll *Polling) onRequest(req *Request) {
	debug("poll.onRequest", req)
	switch req.httpReq.Method {
	case "GET":
		poll.onPollRequest(req)
	case "POST":
		poll.onDataRequest(req)
	default:
		debug("polling default")
		res := req.res
		res.WriteHeader(500)
	}
}

func (poll *Polling) tryWritable(fn, def func()) {
	select {
	case <-poll.readyCh:
		fn()
	default:
		if def != nil {
			def()
		}
	}
}

func (poll *Polling) onPollRequest(req *Request) {
	res := req.res

	if atomic.SwapInt32(&poll.reqGuard, 1) != 0 {
		debug("request overlap")
		poll.onError("overlap from client", "")
		res.WriteHeader(500)
		return
	}
	debug("setting request")

	poll.Emit("drain")

	if poll.shouldClose != nil {
		poll.tryWritable(func() {
			poll.send([]*parser.Packet{&noopPkt})
		}, nil)
	}

	poll.readyCh = make(chan bool)
	poll.writeCh = make(chan []byte, 1)
	timeout := make(chan bool, 1)

	var data []byte = nil
	select {
	case poll.readyCh <- true:
		data = <-poll.writeCh
	case <-timeout:
	}
	poll.doWrite(req, data)
	close(poll.readyCh)
	close(poll.writeCh)
	close(timeout)
	poll.readyCh = nil
	poll.writeCh = nil

	atomic.StoreInt32(&poll.reqGuard, 0)
}

func (poll *Polling) onDataRequest(req *Request) {
	res := req.res

	if atomic.SwapInt32(&poll.dataGuard, 1) != 0 {
		debug("data request overlap from client")
		poll.onError("data request overlap from client", "")
		res.WriteHeader(500)
		return
	}

	chunks := new(bytes.Buffer)
	buffer := make([]byte, 0, 4096)
	for {
		_, err := req.httpReq.Body.Read(buffer)
		if len(buffer)+chunks.Len() > poll.maxHTTPBufferSize {
			chunks.Reset()
			req.httpReq.Body.Close()
			req.httpReq.Close = true
		} else {
			chunks.Write(buffer)
		}
		if err != nil {
			break
		}
	}

	debug("data request onEnd ok")
	go poll.onData(chunks.Next(chunks.Len()))

	res.Header().Set("Content-Length", "2")
	res.Header().Set("Content-Type", "text/html")
	poll.headers(req)
	res.WriteHeader(200)
	res.Write([]byte("ok"))

	atomic.StoreInt32(&poll.dataGuard, 0)
}

func (poll *Polling) onData(data []byte) {
	debug(fmt.Sprintf("received \"%s\"", string(data)))
	parser.DecodePayload(data, func(pkt parser.Packet, index, total int) {
		if pkt.Type == "close" {
			debug("got xhr close packet")
			poll.onClose()
			return
		}
		poll.onPacket(&pkt)
	})
}

func (poll *Polling) send(pkts []*parser.Packet) {
	if poll.shouldClose != nil {
		debug("appending close packet to payload")
		pkts = append(pkts, &parser.Packet{Type: "close"})
		poll.shouldClose()
		poll.shouldClose = nil
	}
	debug("poll.send")
	for _, pkt := range pkts {
		debug(*pkt)
	}

	parser.EncodePayload(pkts, poll.supportsBinary, func(data []byte) {
		poll.write(data)
	})
}

func (poll *Polling) write(data []byte) {
	debug(fmt.Sprintf("writing \"%s\"", string(data)))
	poll.writeCh <- data
}

func (poll *Polling) setMaxHTTPBufferSize(size int) {
	poll.maxHTTPBufferSize = size
}
