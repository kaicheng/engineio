package engineio

import (
	"fmt"
	"net/http"
	"time"
)

var Protocol int = 1

func getPath(opts Options) string {
	if opts == nil || opts["path"] == nil || len(opts["path"].(string)) == 0 {
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
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        srvMux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	srv := Attach(server, opts)
	go server.ListenAndServe()
	return srv
}

func Attach(server *http.Server, opts Options) *Server {
	mux := http.NewServeMux()
	srv := NewServer(opts)
	path := getPath(opts)
	debug(fmt.Sprintf("intercepting request for path \"%s\"", path))
	mux.Handle(path, srv)
	if path != "/" {
		mux.Handle("/", server.Handler)
	}

	server.Handler = mux
	return srv
}
