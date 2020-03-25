package main

import (
	"encoding/hex"
	"errors"
	"io"
)

var noMessage error = errors.New("no message")

type MessageBuffer struct {
	buffer []byte
}

type Message struct {
	buffer []byte
}

// readInt read a 32-bit integer from byte buffer start
// from offset
func readInt(b []byte, offset int) (int, error) {
	if offset < 0 || offset+4 > len(b) {
		return 0, errors.New("out of index when reading integer")
	}
	n := int(b[offset]) & 0xff
	n <<= 8
	n |= int(b[offset+1]) & 0xff
	n <<= 8
	n |= int(b[offset+2]) & 0xff
	n <<= 8
	n |= int(b[offset+3]) & 0xff
	return n, nil
}

func writeInt(b []byte, offset int, value int) error {
	if offset < 0 || offset+4 > len(b) {
		return errors.New("out of index when writting integer")
	}

	b[offset+0] = byte((value >> 24) & 0xff)
	b[offset+1] = byte((value >> 16) & 0xff)
	b[offset+2] = byte((value >> 8) & 0xff)
	b[offset+3] = byte(value & 0xff)
	return nil
}

// NewMessageBuffer creaate a MessageBuffer object
func NewMessageBuffer() *MessageBuffer {
	return &MessageBuffer{buffer: make([]byte, 0)}
}

// Add add data to the buffer
func (p *MessageBuffer) Add(b []byte) {
	p.buffer = append(p.buffer, b...)
}

// ExtractMessage extract a thrift message
func (p *MessageBuffer) ExtractMessage() (*Message, error) {
	if len(p.buffer) > 4 {
		n, err := readInt(p.buffer, 0)
		if err == nil && len(p.buffer) >= 4+n {
			msg := &Message{buffer: p.buffer[0 : 4+n]}
			p.buffer = p.buffer[4+n:]
			return msg, nil
		}
	}
	return nil, noMessage
}

// NewMessage create a thrift Message object
func NewMessage(b []byte) *Message {
	return &Message{buffer: b}
}

// Write write the message to a io.Writer object
func (m *Message) Write(writer io.Writer) error {
	_, err := writer.Write(m.buffer)
	return err
}

// Hex convert the message to hex format
func (m *Message) Hex() string {
	return hex.Dump(m.buffer)
}

func (m *Message) GetSeqId() (int, error) {
	offset, err := m.getSeqIdOffset()
	if err == nil {
		return readInt(m.buffer, offset)
	}

	return 0, err
}

func (m *Message) SetSeqId(seqId int) error {
	offset, err := m.getSeqIdOffset()
	if err == nil {
		return writeInt(m.buffer, offset, seqId)
	}

	return err

}

// GetName get the name of call
func (m *Message) GetName() (string, error) {
	offset := 4
	if m.isFramed() {
		offset += 4
	}
	n, err := readInt(m.buffer, offset)
	if err == nil {
		return string(m.buffer[offset+4 : offset+4+n]), nil
	}
	return "", err
}

// GetType get message type
// - 1, Call
// - 2, Reply
// - 3, Exception
// - 4, Oneway
func (m *Message) GetType() int {
	offset := 0
	if m.isFramed() {
		offset += 4
	}
	return int(m.buffer[offset+3] & 0xff)
}

func (m *Message) isFramed() bool {
	return m.buffer[0]&0x80 != 0x80
}

func (m *Message) getSeqIdOffset() (int, error) {
	offset := 4
	if m.isFramed() {
		offset += 4
	}
	// read the name length
	n, err := readInt(m.buffer, offset)
	if err != nil {
		return 0, err
	}
	return offset + 4 + n, nil
}
