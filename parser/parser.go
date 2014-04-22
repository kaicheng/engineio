package parser

import (
	"bytes"
)

type EncodeCallback func(data []byte)
type DecodeCallback func(pkt Packet) Packet

var errPkg = Packet{Type:"error", Data:[]byte("parser error")}

func EncodePacket(pkt *Packet, callback EncodeCallback) {
	defer func() {
		recover()
	} ()
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
	if (int(t) >= len(PacketsList)) {
		return errPkg
	}

	if (len(data) > 1) {
		newData := make([]byte, len(data) - 1)
		copy(newData, data[1:])
		return Packet{Type:PacketsList[t], Data:newData}
	} else {
		return Packet{Type:PacketsList[t]}
	}
}

func DecodeBase64Packet(data []byte) Packet {
	return errPkg
}