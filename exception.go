package main

import ()

func createInternalErrorException(framed bool, name string, seqId int, errMsg string) *Message {
	b := NewBinaryProtocol(framed)
	b.BeginMessage(name, Exception, seqId)
	b.BeginField(STRING, 1)
	b.WriteString(errMsg)
	b.EndField()
	b.BeginField(I32, 2)
	b.WriteInt32(6)
	b.StopField()
	b.EndField()
	b.EndMessage()
	return b.ToMessage()
}
