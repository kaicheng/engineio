package parser

import (
	"bytes"
	"crypto/rand"
	"runtime/debug"
	"testing"
)

func encode(pkt *Packet, callback EncodeCallback) {
	EncodePacket(pkt, false, callback)
}

var decode = DecodePacket

func encPayload(pkts []*Packet, callback EncodeCallback) {
	EncodePayload(pkts, false, callback)
}

var decPayload = DecodePayload

var encPayloadB = EncodePayloadAsBinary

var decPayloadB = DecodePayloadAsBinary

func packetEqual(a, b *Packet) bool {
	return (a.Type == b.Type) && bytes.Equal(a.Data, b.Data)
}

func expect(t *testing.T, res bool, msgs ...interface{}) {
	if !res {
		debug.PrintStack()
		t.Error(msgs...)
	}
}

var errPkg = Packet{Type: "error", Data: []byte("parser error")}

func TestEncode(t *testing.T) {
	encode(&Packet{Type: "message", Data: []byte("test")}, nil)
}

func TestEncodeDecode(t *testing.T) {
	pkt := Packet{Type: "message", Data: []byte("test")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		expect(t, packetEqual(&pkt, &decPkt), "Decode error:", pkt, decPkt)
	})
}

func TestDecodeOpen(t *testing.T) {
	pkt := Packet{Type: "open", Data: []byte("{\"some\":\"json\"}")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		expect(t, packetEqual(&pkt, &decPkt), "Decode error:", pkt, decPkt)
	})
}

func TestDecodeClose(t *testing.T) {
	pkt := Packet{Type: "close"}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		expect(t, packetEqual(&pkt, &decPkt), "Decode error:", pkt, decPkt)
	})
}

func TestDecodePing(t *testing.T) {
	pkt := Packet{Type: "ping", Data: []byte("1")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		expect(t, packetEqual(&pkt, &decPkt), "Decode error:", pkt, decPkt)
	})
}

func TestDecodePong(t *testing.T) {
	pkt := Packet{Type: "pong", Data: []byte("1")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		expect(t, packetEqual(&pkt, &decPkt), "Decode error:", pkt, decPkt)
	})
}

func TestDecodeMessage(t *testing.T) {
	pkt := Packet{Type: "message", Data: []byte("aaa")}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		expect(t, packetEqual(&pkt, &decPkt), "Decode error:", pkt, decPkt)
	})
}

func TestDecodeUpgrade(t *testing.T) {
	pkt := Packet{Type: "upgrade"}
	encode(&pkt, func(data []byte) {
		decPkt := decode(data)
		expect(t, packetEqual(&pkt, &decPkt), "Decode error:", pkt, decPkt)
	})
}

func TestDecodeErrorHandling(t *testing.T) {
	pkt1 := decode([]byte(":::"))
	expect(t, packetEqual(&errPkg, &pkt1), "Decode err: ", "bad format")
	pkt2 := decode([]byte("94103"))
	expect(t, packetEqual(&errPkg, &pkt2), "Decode err: ", "inexistent types")
}

func TestEncodePayloadBasic(t *testing.T) {
	encPayload([]*Packet{&Packet{Type: "ping"}, &Packet{Type: "post"}}, func(data []byte) {
		expect(t, string(data) == "2:b22:b0", "Encode err")
	})
}

func TestEncodeDecodePayload(t *testing.T) {
	msg := Packet{Type: "message", Data: []byte("a")}
	ping := Packet{Type: "ping"}
	encPayload([]*Packet{&msg}, func(data []byte) {
		decPayload(data, func(packet Packet, index, total int) {
			expect(t, packetEqual(&msg, &packet), "Decode err:", msg, packet)
			expect(t, index+1 == total, "Decode err:", "not last")
		})
	})
	encPayload([]*Packet{&msg, &ping}, func(data []byte) {
		decPayload(data, func(packet Packet, index, total int) {
			isLast := index+1 == total
			if isLast {
				expect(t, packetEqual(&ping, &packet), "Decode err:", ping, packet)
			} else {
				expect(t, packetEqual(&msg, &packet), "Decode err:", msg, packet)
			}
		})
	})
}

func TestEncodeDecodeEmptyPayload(t *testing.T) {
	encPayload([]*Packet{}, func(data []byte) {
		decPayload(data, func(packet Packet, index, total int) {
			t.Error("Should not decode any packet")
		})
	})
}

func TestErrOnBadPayloadFormat(t *testing.T) {
	decPayload([]byte("1!"), func(pkt Packet, index, total int) {
		expect(t, packetEqual(&pkt, &errPkt), "Should get error packet")
		expect(t, index+1 == total, "Should be last")
	})
	decPayload([]byte(""), func(pkt Packet, index, total int) {
		expect(t, packetEqual(&pkt, &errPkt), "Should get error packet")
		expect(t, index+1 == total, "Should be last")
	})
	decPayload([]byte("))"), func(pkt Packet, index, total int) {
		expect(t, packetEqual(&pkt, &errPkt), "Should get error packet")
		expect(t, index+1 == total, "Should be last")
	})
}

func TestErrOnBadPayloadLength(t *testing.T) {
	decPayload([]byte("1:"), func(pkt Packet, index, total int) {
		expect(t, packetEqual(&pkt, &errPkt), "Should get error packet")
		expect(t, index+1 == total, "Should be last")
	})
}

func TestErrOnBadPacketFormat(t *testing.T) {
	decPayload([]byte("3:99:"), func(pkt Packet, index, total int) {
		expect(t, packetEqual(&pkt, &errPkt), "Should get error packet")
		expect(t, index+1 == total, "Should be last")
	})
	decPayload([]byte("1:aa"), func(pkt Packet, index, total int) {
		expect(t, packetEqual(&pkt, &errPkt), "Should get error packet")
		expect(t, index+1 == total, "Should be last")
	})
	decPayload([]byte("1:a2:b"), func(pkt Packet, index, total int) {
		expect(t, packetEqual(&pkt, &errPkt), "Should get error packet")
		expect(t, index+1 == total, "Should be last")
	})
}

func TestSimpleEncodePayloadAsBinary(t *testing.T) {
	encPayloadB([]*Packet{&Packet{Type: "close", Data: []byte{2, 3}}}, func(data []byte) {
		expect(t, bytes.Equal(data, []byte{1, 3, 255, 1, 2, 3}), "EncodePayloadAsBinary error")
	})
}

func TestEncodeDecodePayloadAsBinary(t *testing.T) {
	buf := make([]byte, 123)
	rand.Read(buf)
	pkt0 := Packet{Type: "message", Data: buf}
	pkt1 := Packet{Type: "message", Data: []byte("hello")}
	pkt2 := Packet{Type: "close"}
	encPayloadB([]*Packet{&pkt0, &pkt1, &pkt2}, func(data []byte) {
		decPayloadB(data, func(pkt Packet, index, total int) {
			expect(t, total == 3, "Error in total")
			switch index {
			case 0:
				expect(t, packetEqual(&pkt, &pkt0), "Decode err:", pkt, pkt0)
			case 1:
				expect(t, packetEqual(&pkt, &pkt1), "Decode err:", pkt, pkt1)
			case 2:
				expect(t, packetEqual(&pkt, &pkt2), "Decode err:", pkt, pkt2)
			default:
				t.Error("Error in index")
			}
		})
	})
}
