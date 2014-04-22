package parser

import (
	"bytes"
	"testing"
)

func encode(pkt *Packet, callback EncodeCallback) {
	EncodePacket(pkt, false, callback)
}

var decode = DecodePacket

func packetEqual(a, b *Packet) bool {
	return (a.Type == b.Type) && bytes.Equal(a.Data, b.Data)
}

func TestEncode(t *testing.T) {
	encode(&Packet{Type: "message", Data: []byte("test")}, nil)
}

func TestEncodeDecode(t *testing.T) {
	pkt := Packet{Type: "message", Data: []byte("test")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		if !packetEqual(&pkt, &decPkt) {
			t.Error("Decode error:", pkt, decPkt)
		}
	})
}

func TestDecodeOpen(t *testing.T) {
	pkt := Packet{Type: "open", Data: []byte("{\"some\":\"json\"}")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		if !packetEqual(&pkt, &decPkt) {
			t.Error("Decode error:", pkt, decPkt)
		}
	})
}

func TestDecodeClose(t *testing.T) {
	pkt := Packet{Type: "close"}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		if !packetEqual(&pkt, &decPkt) {
			t.Error("Decode error:", pkt, decPkt)
		}
	})
}

func TestDecodePing(t *testing.T) {
	pkt := Packet{Type: "ping", Data: []byte("1")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		if !packetEqual(&pkt, &decPkt) {
			t.Error("Decode error:", pkt, decPkt)
		}
	})
}

func TestDecodePong(t *testing.T) {
	pkt := Packet{Type: "pong", Data: []byte("1")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		if !packetEqual(&pkt, &decPkt) {
			t.Error("Decode error:", pkt, decPkt)
		}
	})
}

func TestDecodeMessage(t *testing.T) {
	pkt := Packet{Type: "message", Data: []byte("aaa")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		if !packetEqual(&pkt, &decPkt) {
			t.Error("Decode error:", pkt, decPkt)
		}
	})
}

func TestDecodeUpgrade(t *testing.T) {
	pkt := Packet{Type: "upgrade"}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		if !packetEqual(&pkt, &decPkt) {
			t.Error("Decode error:", pkt, decPkt)
		}
	})
}

func TestDecodeErrorHandling(t *testing.T) {
	errPkg := Packet{Type: "error", Data: []byte("parser error")}
	pkt1 := decode([]byte(":::"))
	if !packetEqual(&errPkg, &pkt1) {
		t.Error("Decode err: ", "bad format")
	}
	pkt2 := decode([]byte("94103"))
	if !packetEqual(&errPkg, &pkt2) {
		t.Error("Decode err: ", "inexistent types")
	}
}
