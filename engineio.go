package engineio

import (
	"fmt"
	"net/http"
)

var Protocol int = 1

func getPath(opts Options) string {
	if opts == nil || len(opts["path"].(string)) == 0 {
		return "/engine.io"
	}

	path := opts["path"].(string)
	return path
}

func Listen(port int, opts Options) *Server {
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(501)
		res.Write([]byte("Not Implemented."))
	})
	srv := Attach(opts)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	return srv
}

func Attach(opts Options) *Server {
	srv := NewServer(opts)
	path := getPath(opts)
	http.Handle(path, srv)
	return srv
}
