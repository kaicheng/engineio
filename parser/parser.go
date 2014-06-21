package parser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
)

type EncodeCallback func(data []byte)
type DecodeCallback func(pkt Packet)
type DecodePayloadCallback func(pkt Packet, index, total int)

var errPkt = Packet{Type: "error", Data: []byte("parser error")}

func EncodePacket(pkt *Packet, supportsBinary bool, callback EncodeCallback) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	if !supportsBinary && pkt.IsBin {
		EncodeBase64Packet(pkt, callback)
		return
	}

	buf := new(bytes.Buffer)
	buf.Grow(1 + len(pkt.Data))
	if pkt.IsBin {
		buf.WriteByte(Packets[pkt.Type])
	} else {
		buf.WriteByte(Packets[pkt.Type] + '0')
	}
	buf.Write(pkt.Data)
	callback(buf.Next(buf.Len()))
}

func DecodePacket(data []byte) Packet {
	if data[0] == 'b' {
		return DecodeBase64Packet(data[1:])
	}

	t := data[0]
	if t >= '0' {
		t = t - '0'
	}
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
	if supportsBinary {
		EncodePayloadAsBinary(pkts, callback)
		return
	}

	if len(pkts) == 0 {
		callback([]byte("0:"))
		return
	}

	buf := new(bytes.Buffer)
	estLen := 0
	for _, pkt := range pkts {
		// sample encoded: 102:bxmessage
		// message is in base 64
		estLen += 6 + len(pkt.Data)*2
	}
	buf.Grow(estLen)
	for _, pkt := range pkts {
		EncodePacket(pkt, supportsBinary, func(data []byte) {
			buf.Write([]byte(strconv.FormatInt(int64(len(data)), 10)))
			buf.WriteByte(':')
			buf.Write(data)
		})
	}
	callback(buf.Next(buf.Len()))
}

func DecodePayload(data []byte, callback DecodePayloadCallback) {
	if len(data) == 0 {
		callback(errPkt, 0, 1)
		return
	}
	if int(data[0]) < 0x20 {
		DecodePayloadAsBinary(data, callback)
		return
	}

	for base := 0; base < len(data); {
		work := data[base:]
		colon := bytes.IndexByte(work, ':')
		if colon < 0 {
			callback(errPkt, 0, 1)
			return
		}
		length64, err := strconv.ParseInt(string(work[:colon]), 10, 32)
		length := int(length64)
		if err != nil || colon+length >= len(work) {
			callback(errPkt, 0, 1)
			return
		}
		base += colon + length + 1
		if length > 0 {
			pkt := DecodePacket(work[colon+1 : colon+1+length])
			if pkt.Type == errPkt.Type && bytes.Equal(pkt.Data, errPkt.Data) {
				callback(errPkt, 0, 1)
				return
			}
			callback(pkt, base-1, len(data))
		}
	}
}

func EncodePayloadAsBinary(pkts []*Packet, callback EncodeCallback) {
	if len(pkts) == 0 {
		callback([]byte{})
	}

	buf := new(bytes.Buffer)
	estLen := 0
	for _, pkt := range pkts {
		// Estimated length of buffer
		// 1(binary indicator) + 4(length bytes) + 1(255) + len(pkt.Data)
		estLen += 6 + len(pkt.Data)
	}
	buf.Grow(estLen)
	for _, pkt := range pkts {
		EncodePacket(pkt, true, func(data []byte) {
			buf.WriteByte(1)
			length := len(data)
			bitsBuf := make([]byte, 10)
			bits := 0
			for length > 0 {
				bitsBuf[bits] = byte(length % 10)
				bits++
				length = length / 10
			}
			for i := bits - 1; i >= 0; i-- {
				buf.WriteByte(bitsBuf[i])
			}
			buf.WriteByte(255)
			buf.Write(data)
		})
	}
	callback(buf.Next(buf.Len()))
}

func getInt(data []byte) (res int) {
	if len(data) > 10 {
		return -1
	}
	res = 0
	for i := 0; i < len(data); i++ {
		res = res * 10
		res += int(data[i])
	}
	return
}

func DecodePayloadAsBinary(data []byte, callback DecodePayloadCallback) {
	estTotal := bytes.Count(data, []byte{255})
	buf := make([][]byte, estTotal)
	total := 0
	base := 0
	i := 0
	for base < len(data) {
		work := data[base+1:]
		i255 := bytes.IndexByte(work, 255)
		if i255 < 0 {
			callback(errPkt, 0, 1)
			return
		}
		length := getInt(work[:i255])
		if length <= 0 {
			callback(errPkt, 0, 1)
			return
		}
		if i255+length+1 > len(work) {
			callback(errPkt, 0, 1)
			return
		}
		buf[i] = work[i255+1 : i255+1+length]
		i++
		total++
		// 1 + number length + 1(255) + data length
		base += 1 + i255 + 1 + length
	}
	for index := 0; index < total; index++ {
		b := buf[index]
		callback(DecodePacket(b), index, total)
	}
}
