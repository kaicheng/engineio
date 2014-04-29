package main

import (
	"encoding/json"
	"os"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"github.com/kaicheng/goport/engineio"
)

func getOpts(arg string) map[string]interface{} {
	var v map[string]interface{}
	err := json.Unmarshal([]byte(arg), &v)
	if err != nil {
		var vv interface{}
		err = json.Unmarshal([]byte(arg), &vv)
		return nil
	} else {
		return v
	}
}

func main() {
	fmt.Println(os.Args)
	port, _ := strconv.ParseInt(os.Args[1], 10, 32)
	it := os.Args[2]
	opts := getOpts(os.Args[3])

	srvMux := http.NewServeMux()
	srvMux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(501)
		res.Write([]byte("Not Implemented."))
	})
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", int(port)),
		Handler:        srvMux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	engineio.Attach(server, opts)
	
	switch it {
	case "default":
			server.ListenAndServe()

	default:
		return
	}
	server.ListenAndServe()
}