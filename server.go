package engineio

import (
	"fmt"
	"github.com/kaicheng/goport/events"
	"net/http"
	"net/url"
	"strconv"
)

type Options map[string]interface{}

type Server struct {
	events.EventEmitter

	Clients      map[string]*Socket
	clientsCount int

	pingTimeout       int
	pingInterval      int
	upgradeTimeout    int
	maxHttpBufferSize int
	transports        []string
	allowUpgrades     bool
	allowRequest      func(*Request, func(int, bool))
	cookie            string
}

type Request struct {
	httpReq *http.Request
	query   url.Values
	res     http.ResponseWriter
}

var gi = 0

func generateId() string {
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

	transportsArray := make([]string, len(transports))
	i := 0
	for key, _ := range transports {
		transportsArray[i] = key
		i++
	}

	srv.pingTimeout = valueOrDefault(opts, "pingTimeout", 60000).(int)
	srv.pingInterval = valueOrDefault(opts, "pingInterval", 25000).(int)
	srv.upgradeTimeout = valueOrDefault(opts, "upgradeTimeout", 10000).(int)
	srv.maxHttpBufferSize = valueOrDefault(opts, "maxHttpBufferSize", 100000000).(int)
	srv.transports = valueOrDefault(opts, "transports", transportsArray).([]string)
	srv.allowUpgrades = false // original default true
	srv.allowRequest = opts["allowRequest"].(func(*Request, func(int, bool)))

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
	return []string{}
}

func (srv *Server) verify(req *Request, upgrade bool, fn func(int, bool)) {
	transport := req.query["transport"][0]
	sid := req.query["sid"]

	if _, ok := transports[transport]; !ok {
		fn(UNKNOWN_TRANSPORT, false)
		return
	}

	if sid != nil && len(sid) > 0 {
		client, ok := srv.Clients[sid[0]]
		if !ok {
			fn(UNKNOWN_SID, false)
			return
		}
		if !upgrade && client.transport.Name() != transport {
			fn(BAD_REQUEST, false)
		}
	} else {
		if "GET" != req.httpReq.Method {
			fn(BAD_HANDSHAKE_METHOD, false)
			return
		}
		if srv.allowRequest != nil {
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
}

func (srv *Server) ServeHTTP(res http.ResponseWriter, httpreq *http.Request) {
	req := new(Request)
	req.httpReq = httpreq
	req.query = httpreq.URL.Query()
	req.res = res

	srv.verify(req, false, func(err int, success bool) {
		if !success {
			sendErrorMessage(res, err)
			return
		}

		if len(req.query["sid"]) > 0 {
			srv.Clients[req.query["sid"][0]].transport.onRequest(req)
		} else {
			srv.handshake(req.query["transport"][0], req)
		}
	})
}

func (srv *Server) Close() {
	for _, socket := range srv.Clients {
		socket.close()
	}
}

func (srv *Server) handshake(transportName string, req *Request) {
	defer func() {
		recover()
		sendErrorMessage(req.res, BAD_REQUEST)
	}()

	id := generateId()

	transport := transports[transportName](req)

	if "polling" == transportName {
		// transport.setMaxHttpBufferSize(srv.maxHttpBufferSize)
	}

	if getBool(req.query["b64"]) {
		// transport.setSupportsBinary(false)
	} else {
		// transport.setSupportsBinary(true)
	}

	socket := newSocket(id, srv, transport, req)

	if len(srv.cookie) > 0 {
		//TODO(kaicheng): check if we should modify res.Header
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

	srv.Emit("connection", socket)
}
