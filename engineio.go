package engineio

import (
	"fmt"
	"net/http"
)

var Protocol int = 1

func getPath(opts Options) string {
	if opts == nil || len(opts["path"].(string)) == 0 {
		return "/engine.io/"
	}

	path := opts["path"].(string)
	return path
}

func Listen(port int, opts Options) *Server {
	srvMux := http.NewServeMux()
	srvMux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(501)
		res.Write([]byte("Not Implemented."))
	})
	srv := Attach(srvMux, opts)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), srvMux)
	return srv
}

func Attach(srvMux *http.ServeMux, opts Options) *Server {
	srv := NewServer(opts)
	path := getPath(opts)
	srvMux.Handle(path, srv)
	return srv
}
