package main

import (
	"encoding/json"
	"fmt"
	"github.com/kaicheng/goport/engineio"
	"github.com/kaicheng/goport/engineio/parser"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"time"
)

func expect(res bool, msgs ...interface{}) {
	if res {
		fmt.Printf("[\x1b[32;1mPASS\x1b[0m] ")
	} else {
		fmt.Printf("[\x1b[31;1mFAIL\x1b[0m] ")
	}
	fmt.Println(msgs...)
	if !res {
		debug.PrintStack()
		panic("!expect")
	}
}

func log(msgs ...interface{}) {
	fmt.Printf("[\x1b[33;1mINFO\x1b[0m] ")
	fmt.Println(msgs...)
}

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
	// fmt.Println(os.Args)
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
	case "should trigger a connection event with a Socket":
		eio.On("connection", func(socket *engineio.Socket) {
			expect(socket != nil, "socket != nil")
		})
	case "should open with polling by default":
		eio.On("connection", func(socket *engineio.Socket) {
			expect(socket.Transport.Name() == "polling", "socket.Transport.Name() == \"polling\"")
		})
	case "should allow arbitrary data through query string":
		eio.On("connection", func(socket *engineio.Socket) {
			query := socket.Request.Query
			_, found := query["transport"]
			expect(found, "query should has \"transport\"")
			_, found = query["a"]
			expect(found, "query should has \"a\"")
			a := query.Get("a")
			expect(a == "b", "query[\"a\"] == \"b\"")
		})
	case "should allow data through query string in uri":
		eio.On("connection", func(socket *engineio.Socket) {
			query := socket.Request.Query
			expect(len(query.Get("EIO")) > 0, "EIO to be a string")
			expect(query.Get("a") == "b", "a to be \"b\"")
			expect(query.Get("c") == "d", "c to be \"d\"")
		})
	case "should be able to access non-empty writeBuffer at closing (server)":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("close", func(reason, desc string) {
				expect(len(socket.WriteBuffer) == 1, "len(socket.WriteBuffer) == 1")
				time.AfterFunc(10*time.Millisecond, func() {
					expect(len(socket.WriteBuffer) == 0, "len(socket.WriteBuffer) == 0")
				})
			})
			socket.WriteBuffer = append(socket.WriteBuffer, &parser.Packet{Type: "message", Data: []byte("foo")})
			socket.OnError("")
		})
	case "should trigger on server if the client does not pong":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("close", func(reason, desc string) {
				expect(reason == "ping timeout", "reason == \"ping timeout\"")
			})
		})
	case "should trigger when server closes a client":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("close", func(reason, desc string) {
				expect(reason == "forced close", "reason == \"forced close\"")
			})
			time.AfterFunc(10*time.Millisecond, func() {
				socket.Close()
			})
		})
	case "should trigger when client closes":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("close", func(reason, desc string) {
				expect(reason == "transport close", "reason == \"transport close\"")
			})
		})
	case "should arrive from server to client":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.Send([]byte("a"))
		})
	case "should arrive from server to client (multiple)":
		eio.On("connection", func(socket *engineio.Socket) {
			log("Sending a")
			socket.Send([]byte("a"))
			time.AfterFunc(50*time.Millisecond, func() {
				log("Sending b")
				socket.Send([]byte("b"))
				time.AfterFunc(50*time.Millisecond, func() {
					log("Sending c")
					socket.Send([]byte("c"))
					time.AfterFunc(50*time.Millisecond, func() {
						socket.Close()
					})
				})
			})
		})
	case "should not be receiving data when getting a message longer than maxHttpBufferSize when polling":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("message", func(msg []byte) {
				expect(len(msg) == 0, "should not receiving data")
			})
		})
	case "should receive data when getting a message shorter than maxHttpBufferSize when polling":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("message", func(msg []byte) {
				expect(string(msg) == "a", "should receiving data")
			})
		})
	case "should arrive when binary data sent as Buffer (polling)":
		binaryData := make([]byte, 5)
		for i := 0; i < 5; i += 1 {
			binaryData[i] = byte(i)
		}
		eio.On("connection", func(socket *engineio.Socket) {
			socket.SendBin(binaryData)
		})
	case "default":
	default:
		return
	}
	server.ListenAndServe()
}
