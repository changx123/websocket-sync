package websocket

import (
	"net/http"
	"github.com/gorilla/websocket"
	"time"
)

type BufferPool interface {
	websocket.BufferPool
}

type Upgrader struct {
	websocket.Upgrader

	// HandshakeTimeout specifies the duration for the handshake to complete.
	HandshakeTimeout time.Duration

	// ReadBufferSize and WriteBufferSize specify I/O buffer sizes. If a buffer
	// size is zero, then buffers allocated by the HTTP server are used. The
	// I/O buffer sizes do not limit the size of the messages that can be sent
	// or received.
	ReadBufferSize, WriteBufferSize int

	// WriteBufferPool is a pool of buffers for write operations. If the value
	// is not set, then write buffers are allocated to the connection for the
	// lifetime of the connection.
	//
	// A pool is most useful when the application has a modest volume of writes
	// across a large number of connections.
	//
	// Applications should use a single pool for each unique value of
	// WriteBufferSize.
	WriteBufferPool BufferPool

	// Subprotocols specifies the server's supported protocols in order of
	// preference. If this field is not nil, then the Upgrade method negotiates a
	// subprotocol by selecting the first match in this list with a protocol
	// requested by the client. If there's no match, then no protocol is
	// negotiated (the Sec-Websocket-Protocol header is not included in the
	// handshake response).
	Subprotocols []string

	// Error specifies the function for generating HTTP error responses. If Error
	// is nil, then http.Error is used to generate the HTTP response.
	Error func(w http.ResponseWriter, r *http.Request, status int, reason error)

	// CheckOrigin returns true if the request Origin header is acceptable. If
	// CheckOrigin is nil, then a safe default is used: return false if the
	// Origin request header is present and the origin host is not equal to
	// request Host header.
	//
	// A CheckOrigin function should carefully validate the request origin to
	// prevent cross-site request forgery.
	CheckOrigin func(r *http.Request) bool

	// EnableCompression specify if the server should attempt to negotiate per
	// message compression (RFC 7692). Setting this value to true does not
	// guarantee that compression will be supported. Currently only "no context
	// takeover" modes are supported.
	EnableCompression bool

	ReadChanSize, WriteChanSize int

	isInit bool
}

func (u *Upgrader) Init() {
	u.Upgrader.HandshakeTimeout = u.HandshakeTimeout
	u.Upgrader.ReadBufferSize = u.ReadBufferSize
	u.Upgrader.WriteBufferSize = u.WriteBufferSize
	u.Upgrader.WriteBufferPool = u.WriteBufferPool
	u.Upgrader.Subprotocols = u.Subprotocols
	u.Upgrader.Error = u.Error
	u.Upgrader.CheckOrigin = u.CheckOrigin
	u.Upgrader.EnableCompression = u.EnableCompression
	u.isInit = true
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*Conn, error) {
	if !u.isInit {
		u.Init()
	}
	conn, err := u.Upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}
	outConn := &Conn{Conn: *conn, writeChan: make(chan *writeInfo, u.WriteChanSize), readerChan: make(chan *readerInfo, u.ReadChanSize), closeChan: make(chan byte, 1)}
	go outConn.readerLoop()
	go outConn.writeLoop()
	return outConn, nil
}
