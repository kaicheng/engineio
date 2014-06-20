package engineio

import (
	"fmt"
)

type XHR struct {
	Polling
}

func (xhr *XHR) InitXHR(req *Request) {
	xhr.InitPolling(req)

	xhr.Polling.doWrite = func(data []byte) {
		debug(fmt.Sprintf("xhr writing \"%s\"", string(data)))
		contentType := "text/plains; charset=UTF-8"
		contentLength := fmt.Sprintf("%d", len(data))
		res := req.res
		res.Header().Set("Content-Type", contentType)
		res.Header().Set("Content-Length", contentLength)
		xhr.headers(req)
		res.WriteHeader(200)
		res.Write(data)
	}

	xhr.Polling.headers = func(req *Request) {
		xhr.Emit("headers", req.res.Header())
	}
}

func (xhr *XHR) onRequest(req *Request) {
	if "OPTIONS" == req.httpReq.Method {
		debug("xhr Method == OPTIONS")
		res := req.res
		xhr.headers(req)
		res.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		res.WriteHeader(200)
		res.Write(nil)
	} else {
		xhr.Polling.onRequest(req)
	}
}
