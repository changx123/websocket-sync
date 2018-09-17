package websocket

import (
	"github.com/gorilla/websocket.git"
	"sync"
)

type readerInfo struct {
	messageType int
	p           []byte
	err         error
}

type writeInfo struct {
	messageType int
	data        []byte
}

//重写 线程安全websocket io
type Conn struct {
	websocket.Conn
	writeChan  chan *writeInfo
	readerChan chan *readerInfo
	closeChan  chan byte
	isClosed   bool
	mutex      sync.Mutex
	writeErr   error
}

func (c *Conn) ReadMessage() (messageType int, p []byte, err error) {
	data := <-c.readerChan
	return data.messageType, data.p, data.err
}

func (c *Conn) WriteMessage(messageType int, data []byte) error {
	select {
	case c.writeChan <- &writeInfo{messageType, data}:
	case <-c.closeChan:
		return c.writeErr
	}
	return nil
}

func (c *Conn) Close() {
	c.Conn.Close()
	c.mutex.Lock()
	if !c.isClosed {
		close(c.closeChan)
		c.isClosed = true
	}
	c.mutex.Unlock()
}

func (c *Conn) readerLoop() {
	defer c.Close()
	for {
		messageType, p, err := c.Conn.ReadMessage()
		select {
		case c.readerChan <- &readerInfo{messageType, p, err}:
		case <-c.closeChan:
			return
		}
		if err != nil {
			return
		}
	}
}

func (c *Conn) writeLoop() {
	defer c.Close()
	for v := range c.writeChan {
		err := c.Conn.WriteMessage(v.messageType, v.data)
		if err != nil {
			c.writeErr = err
			break
		}
	}
}
