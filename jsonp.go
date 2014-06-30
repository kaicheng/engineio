package engineio

import (
	"fmt"
)

// FIXME: just space holder for jsonp. Not tested.

type JSONP struct {
	Polling

	head string
	foot string
}

func (jsonp *JSONP) InitJSONP(req *Request) {
	jsonp.InitPolling(req)

	head := fmt.Sprintf("___eio[%s](", req.Query.Get("j"))
	jsonp.head = head
	jsonp.foot = ");"

	jsonp.Polling.doWrite = func(req *Request, data []byte) {
		debug(fmt.Sprintf("jsonp writing \"%s\"", string(data)))

		// FIXME: stringify
		content := jsonp.head + string(data) + jsonp.foot
		contentLength := fmt.Sprintf("%d", len(content))
		res := req.res
		res.Header().Set("Content-Type", "text/javascript; charset=UTF-8")
		res.Header().Set("Content-Length", contentLength)

		// TODO: Prevent XSS warning on IE.

		jsonp.headers(req)
		res.WriteHeader(200)
		res.Write(data)
	}

	jsonp.Polling.headers = func(req *Request) {
		res := req.res

		// TODO: disable XSS protection for IE

		jsonp.Emit("headers", res.Header())
	}
}

func (jsonp *JSONP) onData(data []byte) {
	// TODO: add DoS protection
	jsonp.Polling.onData(data)
}
