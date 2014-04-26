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
		contentType := "text/plains; charset=UTF-8"
		contentLength := fmt.Sprintf("%d", len(data))
		xhr.res.Header().Set("Content-Type", contentType)
		xhr.res.Header().Set("Content-Length", contentLength)
		xhr.headers(req)
		xhr.res.WriteHeader(200)
		xhr.res.Write(data)
	}

	xhr.Polling.headers = func(req *Request) {
		xhr.Emit("headers", req.res.Header())
	}
}

func (xhr *XHR) onRequest(req *Request) {
	if "OPTIONS" == req.httpReq.Method {
		res := req.res
		xhr.headers(req)
		res.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		res.WriteHeader(200)
		res.Write([]byte("TEST"))
	} else {
		xhr.Polling.onRequest(req)
	}
}
