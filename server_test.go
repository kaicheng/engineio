package engineio

import (
	"fmt"
	"net/http"
	"testing"
)

func TestListen(t *testing.T) {
	port := 8088
	Listen(port, nil)
	res, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	t.Log(res, err)
}
