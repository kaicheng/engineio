package engineio

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"runtime/debug"
	"testing"
	"time"
)

func expect(t *testing.T, res bool, msgs ...interface{}) {
	if !res {
		debug.PrintStack()
		t.Error(msgs...)
	}
}

type simpleResponse struct {
	code   int
	header http.Header
	body   string
}

func getResponse(res *http.Response) *simpleResponse {
	sRes := new(simpleResponse)
	sRes.code = res.StatusCode
	sRes.header = res.Header
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	sRes.body = buf.String()
	return sRes
}

var base int32 = 1000
var maxPort int32 = 65535

func getPort() int {
	return int(rand.Int31n(maxPort-base) + base)
}

func sleep(sec float32) {
	time.Sleep(time.Duration(sec * float32(time.Second)))
}

func TestListen(t *testing.T) {
	port := getPort()
	Listen(port, nil)
	sleep(1)
	res, _ := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	sres := getResponse(res)
	t.Log(sres.code, sres.body)
}

func TestTransportUnknown(t *testing.T) {
	port := getPort()
	Listen(port, nil)
	sleep(1)
	res, _ := http.Get(fmt.Sprintf("http://localhost:%d/engine.io/default/", port))
	sres := getResponse(res)
	t.Log(sres.code, sres.body)
}
