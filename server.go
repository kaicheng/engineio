package engineio

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kaicheng/goport/events"

	//	"runtime/debug"
)

type Options map[string]interface{}

type Server struct {
	events.EventEmitter

	Clients      map[string]*Socket
	clientsCount int

	pingTimeout    time.Duration
	pingInterval   time.Duration
	upgradeTimeout time.Duration

	maxHttpBufferSize int
	transports        []string
	allowUpgrades     bool
	allowRequest      func(*Request, func(int, bool))
	cookie            string
}

type Request struct {
	events.EventEmitter

	httpReq *http.Request
	Query   url.Values
	res     http.ResponseWriter

	abort   func() // used by polling
	cleanup func() // used by polling
}

var gi = 0

func generateId() string {
	// FIXME: Need lock
	gi++
	return fmt.Sprintf("%d", gi)
}

func getBool(val []string) bool {
	if len(val) == 0 {
		return false
	}
	res, err := strconv.ParseBool(val[0])
	if err != nil {
		return false
	}
	return res
}

func valueOrDefault(m Options, key string, def interface{}) interface{} {
	if value, ok := m[key]; ok {
		return value
	} else {
		return def
	}
}

func NewServer(opts Options) (srv *Server) {
	srv = new(Server)
	srv.InitEventEmitter()

	srv.Clients = make(map[string]*Socket)
	srv.clientsCount = 0

	transportsArray := make([]interface{}, len(transports))
	i := 0
	for key, _ := range transports {
		transportsArray[i] = key
		i++
	}

	srv.pingTimeout = (time.Duration(valueOrDefault(opts, "pingTimeout", 60000).(int)) * time.Millisecond)
	srv.pingInterval = (time.Duration(valueOrDefault(opts, "pingInterval", 25000).(int)) * time.Millisecond)
	srv.upgradeTimeout = (time.Duration(valueOrDefault(opts, "upgradeTimeout", 10000).(int)) * time.Millisecond)

	srv.maxHttpBufferSize = valueOrDefault(opts, "maxHttpBufferSize", 100000000).(int)
	tmpTransports := valueOrDefault(opts, "transports", transportsArray).([]interface{})
	srv.transports = make([]string, len(tmpTransports))
	for i, v := range tmpTransports {
		srv.transports[i] = v.(string)
	}
	srv.allowUpgrades = valueOrDefault(opts, "allowUpgrades", true).(bool)
	srv.allowRequest = nil
	srv.cookie = valueOrDefault(opts, "cookie", "io").(string)

	return
}

const (
	UNKNOWN_TRANSPORT int = iota
	UNKNOWN_SID
	BAD_HANDSHAKE_METHOD
	BAD_REQUEST
)

var ErrorMessages = []string{"Transport unknown", "Session ID unknown", "Bad handshake method", "Bad request"}

// TODO(kaicheng): allow upgrades.
func (srv *Server) upgrades(transport string) []string {
	if !srv.allowUpgrades {
		return []string{}
	}
	return transportUpgrades[transport]
}

func (srv *Server) verify(req *Request, upgrade bool, fn func(int, bool)) {
	transport := req.Query.Get("transport")
	sid := req.Query.Get("sid")

	if trans, ok := transports[transport]; !ok || trans == nil {
		debug(fmt.Sprintf("unknown transport \"%s\"", transport))
		fn(UNKNOWN_TRANSPORT, false)
		return
	}

	if len(sid) > 0 {
		client, ok := srv.Clients[sid]
		if !ok {
			fn(UNKNOWN_SID, false)
			return
		}
		if !upgrade && client.Transport.Name() != transport {
			debug("bad request: unexpected transport without upgrade")
			fn(BAD_REQUEST, false)
			return
		}
	} else {
		if "GET" != req.httpReq.Method {
			fn(BAD_HANDSHAKE_METHOD, false)
			return
		}
		if srv.allowRequest == nil {
			fn(0, true)
			return
		}
		srv.allowRequest(req, fn)
		return
	}

	fn(0, true)
	return
}

func sendErrorMessage(res http.ResponseWriter, code int) {
	res.Header().Set("Content-type", "application/json")
	res.WriteHeader(400)
	data := fmt.Sprintf("{\"code\":%d,\"message\":\"%s\"}", code, ErrorMessages[code])
	res.Write([]byte(data))
	/*
		fmt.Printf("\x1b[33;1m[%d] %s.\x1b[0m\n", code, ErrorMessages[code])
		debug.PrintStack()
	*/
}

func (srv *Server) ServeHTTP(res http.ResponseWriter, httpreq *http.Request) {
	debug(fmt.Sprintf("handling \"%s\" http request \"%s\"", httpreq.Method, httpreq.RequestURI))
	req := new(Request)
	req.InitEventEmitter()
	req.httpReq = httpreq
	req.Query = httpreq.URL.Query()
	req.res = res

	srv.verify(req, false, func(err int, success bool) {
		if !success {
			debug("sending error message")
			sendErrorMessage(res, err)
			return
		}

		if len(req.Query["sid"]) > 0 {
			debug("setting new request for existing client")
			srv.Clients[req.Query.Get("sid")].Transport.onRequest(req)
		} else {
			srv.handshake(req.Query.Get("transport"), req)
		}
	})
}

func (srv *Server) Close() {
	debug("closing all open clients")
	for _, socket := range srv.Clients {
		socket.Close()
	}
}

func (srv *Server) handshake(transportName string, req *Request) {
	defer func() {
		/*
			if err := recover(); err != nil {
				fmt.Println(err)
				sendErrorMessage(req.res, BAD_REQUEST)
			}
		*/
	}()

	id := generateId()

	debug(fmt.Sprintf("handshaking client \"%s\"", id))

	transport := transports[transportName](req)

	if "polling" == transportName {
		transport.setMaxHTTPBufferSize(srv.maxHttpBufferSize)
	}

	if getBool(req.Query["b64"]) {
		transport.setSupportsBinary(false)
	} else {
		transport.setSupportsBinary(true)
	}

	socket := newSocket(id, srv, transport, req)

	if len(srv.cookie) > 0 {
		transport.On("headers", func(header http.Header) {
			header.Set("Set-Cookie", srv.cookie+"="+id)
		})
	}

	transport.onRequest(req)

	srv.Clients[id] = socket
	srv.clientsCount++

	socket.Once("close", func() {
		delete(srv.Clients, id)
		srv.clientsCount--
	})

	debug("emitting 'connection'")
	srv.Emit("connection", socket)
}
