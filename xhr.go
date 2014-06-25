package engineio

import (
	"fmt"
)

type XHR struct {
	Polling
}

func (xhr *XHR) InitXHR(req *Request) {
	xhr.InitPolling(req)

	xhr.Polling.doWrite = func(req *Request, data []byte) {
		debug(fmt.Sprintf("xhr writing \"%s\"", string(data)))
		contentType := "text/plains; charset=UTF-8"
		if (data[0] < 20) {
			contentType = "application/octet-stream"
		}
		contentLength := fmt.Sprintf("%d", len(data))
		res := req.res
		res.Header().Set("Content-Type", contentType)
		res.Header().Set("Content-Length", contentLength)

		// TODO: Prevent XSS warning on IE.

		xhr.headers(req)
		res.WriteHeader(200)
		res.Write(data)
	}

	xhr.Polling.headers = func(req *Request) {
		httpreq := req.httpReq
		res := req.res

		origin := httpreq.Header.Get("Origin")

		if (len(origin) > 0) {
			res.Header().Set("Access-Control-Allow-Credentials", "true")
			res.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			res.Header().Set("Access-Control-Allow-Origin", "*")
		}

		xhr.Emit("headers", res.Header())
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
