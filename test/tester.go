package main

import (
	"encoding/json"
	"fmt"
	"github.com/kaicheng/goport/engineio"
	"github.com/kaicheng/goport/engineio/parser"
	"net/http"
	"os"
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
		os.Exit(-1)
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
	case "should be able to open with ws directly":
		eio.On("connection", func(socket *engineio.Socket) {
			expect(socket.Transport.Name() == "websocket", "socket name is web socket.")
		})
	case "should open with polling by default":
		eio.On("connection", func(socket *engineio.Socket) {
			expect(socket.Transport.Name() == "polling", "socket.Transport.Name() == \"polling\"")
		})
	case "default to polling when proxy doesn't support websocket":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("message", func(msg []byte) {
				if "echo" == string(msg) {
					socket.Send(msg)
				}
			})
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
	case "should trigger on server even when there is no outstanding polling request":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("close", func(reason, desc string) {
				expect(reason == "ping timeout", "reason is ping timeout")
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
	case "should not trigger with connection: close header":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("message", func(data []byte) {
				expect(string(data) == "test", "msg == test")
				socket.Send([]byte("woot"))
			})
		})
	case "should trigger early with connection `transport close` after missing pong":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("heartbeat", func() {
				time.AfterFunc(20*time.Millisecond,
					func() { socket.Close() })
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
	case "should execute in order when message sent (client) (polling)":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("message", func(msg []byte) {
				socket.Send(msg)
			})
		})
	case "should execute once for each send":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.Send([]byte("a"))
			socket.Send([]byte("b"))
			socket.Send([]byte("c"))
		})
	case "should emit when socket receives packet":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("packet", func(pkt *parser.Packet) {
				expect(pkt.Type == "message", "packet type error")
				expect(string(pkt.Data) == "a", "packet data error")
			})
		})
	case "should emit when receives ping":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("packet", func(pkt *parser.Packet) {
				socket.Close()
				expect(pkt.Type == "ping", "packet type error")
			})
		})
	case "should emit before socket send message":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("packetCreate", func(pkt *parser.Packet) {
				expect(pkt.Type == "message", "packet type error")
				expect(string(pkt.Data) == "a", "packet data error")
			})
			socket.Send([]byte("a"))
		})
	case "should emit before send pong":
		eio.On("connection", func(socket *engineio.Socket) {
			socket.On("packetCreate", func(pkt *parser.Packet) {
				socket.Close()
				expect(pkt.Type == "pong", "packet type error")
			})
		})
	case "should upgrade":
		eio.On("connection", func(socket *engineio.Socket) {
			var lastSent byte = 0
			var lastReceived byte = 0
			upgraded := false
			expect(socket.Request.Query.Get("transport") == "polling", "transport is polling")
			ticker := time.NewTicker(2 * time.Millisecond)
			go func() {
				for {
					<-ticker.C
					lastSent += 1
					socket.Send([]byte(fmt.Sprintf("%d", lastSent)))
					if 50 == lastSent {
						ticker.Stop()
						return
					}
				}
			}()
			socket.On("message", func(msg []byte) {
				fmt.Println("on.message", string(msg))
				lastReceived += 1
				expect(string(msg) == fmt.Sprintf("%d", lastReceived), "msg == lastReceived")
			})
			socket.On("upgrade", func(to string) {
				expect(socket.Request.Query.Get("transport") == "polling", "original transport is polling")
				upgraded = true
				expect(to == "websocket", "upgrade to websocket")
				expect(socket.Transport.Name() == "websocket", "upgraded to websocket")
			})
			socket.On("close", func(reason string) {
				expect(reason == "transport close", "reason is transport close")
				expect(lastSent == 50, "lastSent == 50")
				expect(lastReceived == 50, "lastReceived == 50")
				expect(upgraded, "upgraded == true")
			})
		})
	case "default":
	default:
		return
	}
	go server.ListenAndServe()
	timer := time.NewTimer(20 * time.Second)
	<-timer.C
}
