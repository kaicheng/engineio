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
		return nil
	} else {
		for k, val := range v {
			switch val.(type) {
			case float64:
				v[k] = int(val.(float64))
			}
		}
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
	eio := engineio.Attach(server, opts)
	
	switch it {
	case "on connection":
		eio.On("connection", func (socket *engineio.Socket) {
			fmt.Println("on connection", *socket)
		})
	case "default":
	default:
		return
	}
	server.ListenAndServe()
}