package engineio

import (
	"time"
)

type ticker struct {
	t   *time.Ticker
	c   <-chan time.Time
	end chan bool
}

func newTicker(d time.Duration) *ticker {
	t := new(ticker)
	t.t = time.NewTicker(d)
	t.c = t.t.C
	t.end = make(chan bool, 1)
	return t
}

func (t *ticker) stop() {
	t.t.Stop()
	// Do NOT block if called multiple times.
	select {
	case t.end <- true:
	default:
	}
}
