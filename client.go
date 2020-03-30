package main

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

type Client struct {
	conn             net.Conn
	requestTimeout   time.Duration
	seqIdAllocator   *SeqIdAllocator
	seqIdMapper      *SeqIdMapper
	loadBalancer     LoadBalancer
	responses        chan *Message
	connLostCallback func(*Client)
}

// NewClient create a thrift client side delegation
func NewClient(conn net.Conn,
	requestTimeout time.Duration,
	seqIdAllocator *SeqIdAllocator,
	loadBalancer LoadBalancer,
	connLostCallback func(*Client)) *Client {
	client := &Client{conn: conn,
		requestTimeout:   requestTimeout,
		seqIdAllocator:   seqIdAllocator,
		seqIdMapper:      NewSeqIdMapper(),
		loadBalancer:     loadBalancer,
		responses:        make(chan *Message, 1000),
		connLostCallback: connLostCallback}

	go client.startReadRequest()
	go client.startWriteResponse()

	return client
}

func (c *Client) startReadRequest() {
	b := make([]byte, 4096)
	buffer := NewMessageBuffer()
	for {
		n, err := c.conn.Read(b)
		if err != nil {
			log.WithFields(log.Fields{"client": c.conn.RemoteAddr().String()}).Error("Lost connection with client")
			close(c.responses)
			c.connLostCallback(c)
			break
		}
		if n > 0 {
			buffer.Add(b[0:n])
			c.processRequestBuffer(buffer)
		}
	}
	log.WithFields(log.Fields{"client": c.conn.RemoteAddr().String()}).Info("Exit read routine")
}

func (c *Client) startWriteResponse() {
	for {
		exitLoop := false
		select {
		case response, ok := <-c.responses:
			if !ok {
				exitLoop = true
				break
			}
			err := response.Write(c.conn)
			if err != nil {
				log.WithFields(log.Fields{"client": c.conn.RemoteAddr().String()}).Error("Fail to send the response")
				exitLoop = true
				break
			}
		}
		if exitLoop {
			break
		}
	}
	log.WithFields(log.Fields{"client": c.conn.RemoteAddr().String()}).Info("Exit write routine")
}

func (c *Client) processRequestBuffer(buffer *MessageBuffer) {
	for {
		request, err := buffer.ExtractMessage()
		if err != nil {
			break
		}
		c.processRequest(request)
	}

}

func (c *Client) processRequest(request *Message) {
	newSeqId, err := c.resetSeqId(request)
	name, _ := request.GetName()
	if err == nil {
		c.loadBalancer.Send(request, time.Now().Add(c.requestTimeout), func(response *Message, err error) {
			c.processResponse(name, newSeqId, request.isFramed(), response, err)
		})
	} else {
		log.WithFields(log.Fields{"error": err}).Error("Fail to send request")
		c.processResponse(name, newSeqId, request.isFramed(), nil, errors.New("No backend servers are available"))
	}
}

func (c *Client) processResponse(name string, newSeqId int, framed bool, response *Message, err error) {

	oldSeqId, ok := c.seqIdMapper.RemoveMap(newSeqId)

	if !ok {
		log.WithFields(log.Fields{"seqId": newSeqId}).Error("Fail to find old seqId")
		return
	}

	if err != nil {
		log.WithFields(log.Fields{"newSeqId": newSeqId, "error": err.Error()}).Error("Fail to send request")
		response = createInternalErrorException(framed, name, oldSeqId, err.Error())
	}

	response.SetSeqId(oldSeqId)

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	c.responses <- response
}

func (c *Client) resetSeqId(request *Message) (int, error) {
	oldSeqId, err := request.GetSeqId()
	if err != nil {
		return 0, err
	}
	newSeqId := c.seqIdAllocator.AllocId()
	err = request.SetSeqId(newSeqId)
	if err != nil {
		return 0, err
	}
	c.seqIdMapper.MapTo(oldSeqId, newSeqId)
	return newSeqId, nil
}

func (c *Client) remoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
