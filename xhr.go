package engineio

type XHR struct {
	Polling
}

func (xhr *XHR) InitXHR(req *Request) {
	xhr.InitPolling(req)
	xhr.Polling.doWrite = func(data []byte) {
		// FIXME
	}
}

func (xhr *XHR) onRequest(req *Request) {
	if "OPTIONS" == req.httpReq.Method {
		res := req.res
		xhr.headers(req)
		res.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		res.WriteHeader(200)
		res.Write(nil)
	} else {
		xhr.Polling.onRequest(req)
	}
}

func (xhr *XHR) doWrite(data []byte) {

}

func (xhr *XHR) headers(req *Request) {
	/* FIXME
	if req.httpReq.Origin {
		req.res.Header().Set("Access-Control-Allow-Credentials", "true")
		// FIXME req.res.Header().Set("Access-Control-Allow-Origin", req.httpReq.Origin)
	} else {
		req.res.Header().Set("Access-Control-Allow-Origin", "*")
	}
	*/

	xhr.Emit("headers", req.res.Header)
}
