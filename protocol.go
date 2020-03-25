package main

import (
	"bytes"
)

type MessageType int
type FieldType int

const (
	Call      MessageType = 1
	Reply                 = 2
	Exception             = 3
	Oneway                = 4
)

const (
	BOOL   FieldType = 2
	BYTE             = 3
	DOUBLE           = 4
	I16              = 6
	I32              = 8
	I64              = 10
	STRING           = 11
	STRUCT           = 12
	MAP              = 13
	SET              = 14
	LIST             = 15
)

type BinaryProtocol struct {
	framed bool
	buf    *bytes.Buffer
}

func NewBinaryProtocol(framed bool) *BinaryProtocol {
	b := &BinaryProtocol{framed: framed, buf: bytes.NewBuffer(make([]byte, 0))}
	b.WriteInt32(0)
	return b
}

func (bp *BinaryProtocol) BeginMessage(name string, msgType MessageType, seqId int) {
	bp.WriteInt32(int(0x80010000 | msgType))
	bp.WriteString(name)
	bp.WriteInt32(seqId)
}

func (bp *BinaryProtocol) EndMessage() {
}

func (bp *BinaryProtocol) BeginStruct() {
	bp.buf.WriteByte(STRUCT)
}

func (bp *BinaryProtocol) EndStruct() {
}

func (bp *BinaryProtocol) BeginField(fieldType FieldType, fieldId int) {
	bp.buf.WriteByte(byte(fieldType))
	bp.buf.WriteByte(byte((fieldId >> 16) & 0xff))
	bp.buf.WriteByte(byte(fieldId & 0xff))
}

func (bp *BinaryProtocol) EndField() {
}

func (bp *BinaryProtocol) StopField() {
	bp.buf.WriteByte(0)
}

func (bp *BinaryProtocol) WriteInt32(value int) {
	bp.buf.WriteByte(byte((value >> 24) & 0xff))
	bp.buf.WriteByte(byte((value >> 16) & 0xff))
	bp.buf.WriteByte(byte((value >> 8) & 0xff))
	bp.buf.WriteByte(byte(value & 0xff))
}

func (bp *BinaryProtocol) WriteString(s string) {
	b := []byte(s)
	bp.WriteBytes(b)
}

func (bp *BinaryProtocol) WriteBytes(b []byte) {
	bp.WriteInt32(len(b))
	bp.buf.Write(b)
}

func (bp *BinaryProtocol) ToMessage() *Message {
	b := bp.buf.Bytes()
	if bp.framed {
		writeInt(b, 0, len(b)-4)
	}
	return NewMessage(b)
}
