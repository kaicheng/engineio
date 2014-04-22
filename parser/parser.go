package parser

import (
	"bytes"
	"encoding/base64"
	"strconv"
)

type EncodeCallback func(data []byte)
type DecodeCallback func(pkt Packet)

var errPkt = Packet{Type: "error", Data: []byte("parser error")}

func EncodePacket(pkt *Packet, supportsBinary bool, callback EncodeCallback) {
	defer func() {
		recover()
	}()

	if !supportsBinary {
		EncodeBase64Packet(pkt, callback)
		return
	}

	buf := new(bytes.Buffer)
	buf.Grow(1 + len(pkt.Data))
	buf.WriteByte(Packets[pkt.Type])
	buf.Write(pkt.Data)
	callback(buf.Next(buf.Len()))
}

func DecodePacket(data []byte) Packet {
	if data[0] == 'b' {
		return DecodeBase64Packet(data[1:])
	}

	t := data[0]
	if int(t) >= len(PacketsList) {
		return errPkt
	}

	if len(data) > 1 {
		newData := make([]byte, len(data)-1)
		copy(newData, data[1:])
		return Packet{Type: PacketsList[t], Data: newData}
	} else {
		return Packet{Type: PacketsList[t]}
	}
}

func EncodeBase64Packet(pkt *Packet, callback EncodeCallback) {
	buf := new(bytes.Buffer)
	buf.Grow(2 + len(pkt.Data)*2)
	buf.WriteByte('b')
	buf.Write([]byte(strconv.FormatInt(int64(Packets[pkt.Type]), 10)))
	buf.Write([]byte(base64.StdEncoding.EncodeToString(pkt.Data)))
	callback(buf.Next(buf.Len()))
}

func DecodeBase64Packet(data []byte) Packet {
	t, err := strconv.ParseUint(string(data[0:1]), 10, 8)
	if err != nil {
		return errPkt
	}
	if int(t) >= len(PacketsList) {
		return errPkt
	}
	dec, err := base64.StdEncoding.DecodeString(string(data[1:]))
	if err != nil {
		return errPkt
	}
	return Packet{Type: PacketsList[t], Data: dec}
}

func EncodePayload(pkts []*Packet, supportsBinary bool, callback EncodeCallback) {
	panic("not implemented")
}

func DecodePayload(data []byte, callback DecodeCallback) {
	panic("not implemented")
}

func EncodePayloadAsBinary(pkts []*Packet, callback EncodeCallback) {
	panic("not implemented")
}

func DecodePayloadAsBinary(data []byte, callback DecodeCallback) {
	panic("not implemented")
}
