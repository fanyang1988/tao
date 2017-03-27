package tao

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"
)

const (
	// MessageTypeBytes is the length of type header.
	MessageTypeBytes = 4
	// MessageLenBytes is the length of length header.
	MessageLenBytes = 4
	// MessageMaxBytes is the maximum bytes allowed for application data.
	MessageMaxBytes = 1 << 23 // 8M
)

// MessageHandler is a combination of message and its handler function.
type MessageHandler struct {
	message Message
	handler HandlerFunc
}

// WriteCloser is the interface that groups Write and Close methods.
type WriteCloser interface {
	Write(Message) error
	WriteByRes(Message) error
	Close()
}

// ServerConn represents a server connection to a TCP server, it implments Conn.
type ServerConn struct {
	netid   int64
	belong  *Server
	rawConn net.Conn

	once      *sync.Once
	wg        *sync.WaitGroup
	sendCh    chan writeData
	handlerCh chan MessageHandler
	timerCh   chan *OnTimeOut

	mu      sync.Mutex // guards following
	name    string
	heart   int64
	pending []int64
	ctx     context.Context
	cancel  context.CancelFunc
	logger LoggerInterface
}

// NewServerConn returns a new server connection which has not started to
// serve requests yet.
func NewServerConn(id int64, s *Server, c net.Conn) *ServerConn {
	sc := &ServerConn{
		netid:     id,
		belong:    s,
		rawConn:   c,
		once:      &sync.Once{},
		wg:        &sync.WaitGroup{},
		sendCh:    make(chan writeData, 1024),
		handlerCh: make(chan MessageHandler, 1024),
		timerCh:   make(chan *OnTimeOut, 1024),
		heart:     time.Now().UnixNano(),
		logger:s.logger,
	}
	sc.ctx, sc.cancel = context.WithCancel(context.WithValue(s.ctx, serverCtx, s))
	sc.name = c.RemoteAddr().String()
	sc.pending = []int64{}
	return sc
}

// ServerFromContext returns the server within the context.
func ServerFromContext(ctx context.Context) (*Server, bool) {
	server, ok := ctx.Value(serverCtx).(*Server)
	return server, ok
}

// GetNetID returns net ID of server connection.
func (sc *ServerConn) GetNetID() int64 {
	return sc.netid
}

// SetName sets name of server connection.
func (sc *ServerConn) SetName(name string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.name = name
}

// GetName returns the name of server connection.
func (sc *ServerConn) GetName() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	name := sc.name
	return name
}

// SetHeartBeat sets the heart beats of server connection.
func (sc *ServerConn) SetHeartBeat(heart int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.heart = heart
}

// GetHeartBeat returns the heart beats of server connection.
func (sc *ServerConn) GetHeartBeat() int64 {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	heart := sc.heart
	return heart
}

// SetContextValue sets extra data to server connection.
func (sc *ServerConn) SetContextValue(k, v interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.ctx = context.WithValue(sc.ctx, k, v)
}

// GetContextValue gets extra data from server connection.
func (sc *ServerConn) GetContextValue(k interface{}) interface{} {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.ctx.Value(k)
}

// Start starts the server connection, creating go-routines for reading,
// writing and handlng.
func (sc *ServerConn) Start() {
	if sc.logger != nil {
		sc.logger.Infof("conn start, <%v -> %v>\n",
			sc.rawConn.LocalAddr(), sc.rawConn.RemoteAddr())
	}
	onConnect := sc.belong.opts.onConnect
	if onConnect != nil {
		onConnect(sc)
	}

	loopers := []func(WriteCloser, *sync.WaitGroup){readLoop, writeLoop, handleLoop}
	for _, l := range loopers {
		looper := l
		sc.wg.Add(1)
		go looper(sc, sc.wg)
	}
}

// Close gracefully closes the server connection. It blocked until all sub
// go-routines are completed and returned.
func (sc *ServerConn) Close() {
	sc.once.Do(func() {
		if sc.logger != nil {
			sc.logger.Infof("conn close gracefully, <%v -> %v>\n",
				sc.rawConn.LocalAddr(), sc.rawConn.RemoteAddr())
		}

		// callback on close
		onClose := sc.belong.opts.onClose
		if onClose != nil {
			onClose(sc)
		}

		// remove connection from server
		sc.logger.Tracef("remove %v", sc.netid)
		sc.belong.conns.Remove(sc.netid)
		addTotalConn(-1)

		// close net.Conn, any blocked read or write operation will be unblocked and
		// return errors.
		if tc, ok := sc.rawConn.(*net.TCPConn); ok {
			// avoid time-wait state
			tc.SetLinger(0)
		}
		sc.rawConn.Close()

		// cancel readLoop, writeLoop and handleLoop go-routines.
		sc.mu.Lock()
		sc.cancel()
		pending := sc.pending
		sc.pending = nil
		sc.mu.Unlock()

		// clean up pending timers
		for _, id := range pending {
			sc.CancelTimer(id)
		}

		// wait until all go-routines exited.
		sc.wg.Wait()

		// close all channels and block until all go-routines exited.
		close(sc.sendCh)
		close(sc.handlerCh)
		close(sc.timerCh)

		// tell server I'm done.
		sc.belong.wg.Done()
	})
}

// AddPendingTimer adds a timer ID to server Connection.
func (sc *ServerConn) AddPendingTimer(timerID int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.pending != nil {
		sc.pending = append(sc.pending, timerID)
	}
}

// Write writes a message to the client.
func (sc *ServerConn) Write(message Message) error {
	return asyncWrite(sc, message, nil)
}

func (cc *ServerConn) WriteByRes(message Message) error {
	resChan := make(chan bool, 1)
	err := asyncWrite(cc, message, resChan)
	if err != nil{
		return err
	}

	<-resChan
	return nil
}

// RunAt runs a callback at the specified timestamp.
func (sc *ServerConn) RunAt(timestamp time.Time, callback func(time.Time, WriteCloser)) int64 {
	id := runAt(sc.ctx, sc.netid, sc.belong.timing, timestamp, callback)
	if id >= 0 {
		sc.AddPendingTimer(id)
	}
	return id
}

// RunAfter runs a callback right after the specified duration ellapsed.
func (sc *ServerConn) RunAfter(duration time.Duration, callback func(time.Time, WriteCloser)) int64 {
	id := runAfter(sc.ctx, sc.netid, sc.belong.timing, duration, callback)
	if id >= 0 {
		sc.AddPendingTimer(id)
	}
	return id
}

// RunEvery runs a callback on every interval time.
func (sc *ServerConn) RunEvery(interval time.Duration, callback func(time.Time, WriteCloser)) int64 {
	id := runEvery(sc.ctx, sc.netid, sc.belong.timing, interval, callback)
	if id >= 0 {
		sc.AddPendingTimer(id)
	}
	return id
}

// CancelTimer cancels a timer with the specified ID.
func (sc *ServerConn) CancelTimer(timerID int64) {
	cancelTimer(sc.belong.timing, timerID)
}

func cancelTimer(timing *TimingWheel, timerID int64) {
	if timing != nil {
		timing.CancelTimer(timerID)
	}
}

// RemoteAddr returns the remote address of server connection.
func (sc *ServerConn) RemoteAddr() net.Addr {
	return sc.rawConn.RemoteAddr()
}

// LocalAddr returns the local address of server connection.
func (sc *ServerConn) LocalAddr() net.Addr {
	return sc.rawConn.LocalAddr()
}

// ClientConn represents a client connection to a TCP server.
type ClientConn struct {
	addr      string
	opts      options
	netid     int64
	rawConn   net.Conn
	once      *sync.Once
	wg        *sync.WaitGroup
	sendCh    chan writeData
	handlerCh chan MessageHandler
	timing    *TimingWheel
	mu        sync.Mutex // guards following
	name      string
	heart     int64
	pending   []int64
	ctx       context.Context
	cancel    context.CancelFunc
	logger LoggerInterface
}

// NewClientConn returns a new client connection which has not started to
// serve requests yet.
func NewClientConn(netid int64, c net.Conn, opt ...ServerOption) *ClientConn {
	var opts options
	for _, o := range opt {
		o(&opts)
	}
	if opts.codec == nil {
		opts.codec = TypeLengthValueCodec{}
	}
	return newClientConnWithOptions(netid, c, opts)
}

func newClientConnWithOptions(netid int64, c net.Conn, opts options) *ClientConn {
	cc := &ClientConn{
		addr:      c.RemoteAddr().String(),
		opts:      opts,
		netid:     netid,
		rawConn:   c,
		once:      &sync.Once{},
		wg:        &sync.WaitGroup{},
		sendCh:    make(chan writeData, 1024),
		handlerCh: make(chan MessageHandler, 1024),
		heart:     time.Now().UnixNano(),
	}
	cc.ctx, cc.cancel = context.WithCancel(context.Background())
	cc.timing = NewTimingWheel(cc.ctx)
	cc.name = c.RemoteAddr().String()
	cc.pending = []int64{}
	return cc
}

// GetNetID returns the net ID of client connection.
func (cc *ClientConn) GetNetID() int64 {
	return cc.netid
}

// SetName sets the name of client connection.
func (cc *ClientConn) SetName(name string) {
	cc.mu.Lock()
	cc.name = name
	cc.mu.Unlock()
}

// GetName gets the name of client connection.
func (cc *ClientConn) GetName() string {
	cc.mu.Lock()
	name := cc.name
	cc.mu.Unlock()
	return name
}

// SetHeartBeat sets the heart beats of client connection.
func (cc *ClientConn) SetHeartBeat(heart int64) {
	cc.mu.Lock()
	cc.heart = heart
	cc.mu.Unlock()
}

// GetHeartBeat gets the heart beats of client connection.
func (cc *ClientConn) GetHeartBeat() int64 {
	cc.mu.Lock()
	heart := cc.heart
	cc.mu.Unlock()
	return heart
}

// SetContextValue sets extra data to client connection.
func (cc *ClientConn) SetContextValue(k, v interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.ctx = context.WithValue(cc.ctx, k, v)
}

// GetContextValue gets extra data from client connection.
func (cc *ClientConn) GetContextValue(k interface{}) interface{} {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.ctx.Value(k)
}

// Start starts the client connection, creating go-routines for reading,
// writing and handlng.
func (cc *ClientConn) Start() {
	if cc.logger != nil {
		cc.logger.Infof("conn start, <%v -> %v>\n",
			cc.rawConn.LocalAddr(), cc.rawConn.RemoteAddr())
	}
	onConnect := cc.opts.onConnect
	if onConnect != nil {
		onConnect(cc)
	}

	loopers := []func(WriteCloser, *sync.WaitGroup){readLoop, writeLoop, handleLoop}
	for _, l := range loopers {
		looper := l
		cc.wg.Add(1)
		go looper(cc, cc.wg)
	}
}

// Close gracefully closes the client connection. It blocked until all sub
// go-routines are completed and returned.
func (cc *ClientConn) Close() {
	cc.once.Do(func() {
		if cc.logger != nil {
			cc.logger.Infof("conn close gracefully, <%v -> %v>\n",
				cc.rawConn.LocalAddr(), cc.rawConn.RemoteAddr())
		}

		// callback on close
		onClose := cc.opts.onClose
		if onClose != nil {
			onClose(cc)
		}

		// close net.Conn, any blocked read or write operation will be unblocked and
		// return errors.
		cc.rawConn.Close()

		// cancel readLoop, writeLoop and handleLoop go-routines.
		cc.mu.Lock()
		cc.cancel()
		cc.pending = nil
		cc.mu.Unlock()

		// stop timer
		cc.timing.Stop()

		// wait until all go-routines exited.
		cc.wg.Wait()

		// close all channels.
		close(cc.sendCh)
		close(cc.handlerCh)

		// cc.once is a *sync.Once. After reconnect() returned, cc.once will point
		// to a newly-allocated one while other go-routines such as readLoop,
		// writeLoop and handleLoop blocking on the old *sync.Once continue to
		// execute Close() (and of course do nothing because of sync.Once).
		// NOTE that it will cause an "unlock of unlocked mutex" error if cc.once is
		// a sync.Once struct, because "defer o.m.Unlock()" in sync.Once.Do() will
		// be performed on an unlocked mutex(the newly-allocated one noticed above)
		if cc.opts.reconnect {
			cc.reconnect()
		}
	})
}

// reconnect reconnects and returns a new *ClientConn.
func (cc *ClientConn) reconnect() {
	var c net.Conn
	var err error
	if cc.opts.tlsCfg != nil {
		c, err = tls.Dial("tcp", cc.addr, cc.opts.tlsCfg)
		if err != nil {
			if cc.logger != nil {
				cc.logger.Criticalf("tls dial error", err)
			}
		}
	} else {
		c, err = net.Dial("tcp", cc.addr)
		if err != nil {
			if cc.logger != nil {
				cc.logger.Criticalf("net dial error", err)
			}
		}
	}
	// copy the newly-created *ClientConn to cc, so after
	// reconnect returned cc will be updated to new one.
	*cc = *newClientConnWithOptions(cc.netid, c, cc.opts)
	cc.Start()
}

// Write writes a message to the client.
func (cc *ClientConn) Write(message Message) error {
	return asyncWrite(cc, message, nil)
}

func (cc *ClientConn) WriteByRes(message Message) error {
	resChan := make(chan bool, 1)
	err := asyncWrite(cc, message, resChan)
	if err != nil{
		return err
	}

	<-resChan
	return nil
}

// RunAt runs a callback at the specified timestamp.
func (cc *ClientConn) RunAt(timestamp time.Time, callback func(time.Time, WriteCloser)) int64 {
	id := runAt(cc.ctx, cc.netid, cc.timing, timestamp, callback)
	if id >= 0 {
		cc.AddPendingTimer(id)
	}
	return id
}

// RunAfter runs a callback right after the specified duration ellapsed.
func (cc *ClientConn) RunAfter(duration time.Duration, callback func(time.Time, WriteCloser)) int64 {
	id := runAfter(cc.ctx, cc.netid, cc.timing, duration, callback)
	if id >= 0 {
		cc.AddPendingTimer(id)
	}
	return id
}

// RunEvery runs a callback on every interval time.
func (cc *ClientConn) RunEvery(interval time.Duration, callback func(time.Time, WriteCloser)) int64 {
	id := runEvery(cc.ctx, cc.netid, cc.timing, interval, callback)
	if id >= 0 {
		cc.AddPendingTimer(id)
	}
	return id
}

// AddPendingTimer adds a new timer ID to client connection.
func (cc *ClientConn) AddPendingTimer(timerID int64) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if cc.pending != nil {
		cc.pending = append(cc.pending, timerID)
	}
}

// CancelTimer cancels a timer with the specified ID.
func (cc *ClientConn) CancelTimer(timerID int64) {
	cancelTimer(cc.timing, timerID)
}

// RemoteAddr returns the remote address of server connection.
func (cc *ClientConn) RemoteAddr() net.Addr {
	return cc.rawConn.RemoteAddr()
}

// LocalAddr returns the local address of server connection.
func (cc *ClientConn) LocalAddr() net.Addr {
	return cc.rawConn.LocalAddr()
}

func runAt(ctx context.Context, netID int64, timing *TimingWheel, ts time.Time, cb func(time.Time, WriteCloser)) int64 {
	timeout := NewOnTimeOut(NewContextWithNetID(ctx, netID), cb)
	return timing.AddTimer(ts, 0, timeout)
}

func runAfter(ctx context.Context, netID int64, timing *TimingWheel, d time.Duration, cb func(time.Time, WriteCloser)) int64 {
	delay := time.Now().Add(d)
	return runAt(ctx, netID, timing, delay, cb)
}

func runEvery(ctx context.Context, netID int64, timing *TimingWheel, d time.Duration, cb func(time.Time, WriteCloser)) int64 {
	delay := time.Now().Add(d)
	timeout := NewOnTimeOut(NewContextWithNetID(ctx, netID), cb)
	return timing.AddTimer(delay, d, timeout)
}

func asyncWrite(c interface{}, m Message, cd chan bool) error {
	defer func() error {
		if p := recover(); p != nil {
			return ErrServerClosed
		}
		return nil
	}()

	var (
		pkt    []byte
		err    error
		sendCh chan writeData
	)
	switch c := c.(type) {
	case *ServerConn:
		pkt, err = c.belong.opts.codec.Encode(m)
		sendCh = c.sendCh

	case *ClientConn:
		pkt, err = c.opts.codec.Encode(m)
		sendCh = c.sendCh
	}

	if err != nil {
		return err
	}

	select {
	case sendCh <- writeData{
		data:pkt,
		cbRes:cd,
	}:
		return nil
	default:
		return ErrWouldBlock
	}
}

/* readLoop() blocking read from connection, deserialize bytes into message,
then find corresponding handler, put it into channel */
func readLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		rawConn          net.Conn
		codec            Codec
		cDone            <-chan struct{}
		sDone            <-chan struct{}
		setHeartBeatFunc func(int64)
		onMessage        onMessageFunc
		handlerCh        chan MessageHandler
		msg              Message
		err              error
		logger LoggerInterface
	)

	switch c := c.(type) {
	case *ServerConn:
		rawConn = c.rawConn
		codec = c.belong.opts.codec
		cDone = c.ctx.Done()
		sDone = c.belong.ctx.Done()
		setHeartBeatFunc = c.SetHeartBeat
		onMessage = c.belong.opts.onMessage
		handlerCh = c.handlerCh
		logger = c.logger
	case *ClientConn:
		rawConn = c.rawConn
		codec = c.opts.codec
		cDone = c.ctx.Done()
		sDone = nil
		setHeartBeatFunc = c.SetHeartBeat
		onMessage = c.opts.onMessage
		handlerCh = c.handlerCh
	}

	defer func() {
		if p := recover(); p != nil {
			if logger != nil {
				logger.Errorf("panics: %v\n", p)
			}
		}
		wg.Done()
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connection closed
			if logger != nil {
				logger.Debug("receiving cancel signal from conn")
			}
			return
		case <-sDone: // server closed
			if logger != nil {
				logger.Debug("receiving cancel signal from server")
			}
			return
		default:
			msg, err = codec.Decode(rawConn)
			if err != nil {
				if logger != nil {
					logger.Errorf("error decoding message %v\n", err)
				}
				if _, ok := err.(ErrUndefined); ok {
					// update heart beats
					setHeartBeatFunc(time.Now().UnixNano())
					continue
				}
				return
			}
			setHeartBeatFunc(time.Now().UnixNano())
			handler := GetHandlerFunc(msg.MessageNumber())
			if handler == nil {
				if onMessage != nil {
					if logger != nil {
						logger.Infof("message %d call onMessage()\n", msg.MessageNumber())
					}
					onMessage(msg, c.(WriteCloser))
				} else {
					if logger != nil {
						logger.Warnf("no handler or onMessage() found for message %d\n",
							msg.MessageNumber())
					}
				}
				continue
			}
			handlerCh <- MessageHandler{msg, handler}
		}
	}
}

type writeData struct {
	data []byte
	cbRes chan bool
}

/* writeLoop() receive message from channel, serialize it into bytes,
then blocking write into connection */
func writeLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		rawConn net.Conn
		sendCh  chan writeData
		cDone   <-chan struct{}
		sDone   <-chan struct{}
		err     error
		logger LoggerInterface
	)

	switch c := c.(type) {
	case *ServerConn:
		rawConn = c.rawConn
		sendCh = c.sendCh
		cDone = c.ctx.Done()
		sDone = c.belong.ctx.Done()
		logger = c.logger
	case *ClientConn:
		rawConn = c.rawConn
		sendCh = c.sendCh
		cDone = c.ctx.Done()
		sDone = nil
	}

	defer func() {
		if p := recover(); p != nil {
			if logger!= nil {
				logger.Errorf("panics: %v\n", p)
			}
		}
		// drain all pending messages before exit
	OuterFor:
		for {
			select {
			case pkt := <-sendCh:
				if pkt.data != nil {
					if _, err = rawConn.Write(pkt.data); err != nil {
						if logger!= nil {
							logger.Errorf("error writing data %v\n", err)
						}
					}else{
						if pkt.cbRes != nil{
							pkt.cbRes <- true
						}
					}
				}
			default:
				break OuterFor
			}
		}
		wg.Done()
		if logger != nil {
			logger.Debug("writeLoop go-routine exited")
		}
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connection closed
			if logger != nil {
				logger.Debug("receiving cancel signal from conn")
			}
			return
		case <-sDone: // server closed
			if logger != nil {
				logger.Debug("receiving cancel signal from server")
			}
			return
		case pkt := <-sendCh:
			if pkt.data != nil {
				if _, err = rawConn.Write(pkt.data); err != nil {
					if logger != nil {
						logger.Errorf("error writing data %v\n", err)
					}
					return
				}else{
					if pkt.cbRes != nil{
						pkt.cbRes <- true
					}
				}
			}
		}
	}
}

// handleLoop() - put handler or timeout callback into worker go-routines
func handleLoop(c WriteCloser, wg *sync.WaitGroup) {
	var (
		cDone        <-chan struct{}
		sDone        <-chan struct{}
		timerCh      chan *OnTimeOut
		handlerCh    chan MessageHandler
		netID        int64
		ctx          context.Context
		askForWorker bool
		logger LoggerInterface
	)

	switch c := c.(type) {
	case *ServerConn:
		cDone = c.ctx.Done()
		sDone = c.belong.ctx.Done()
		timerCh = c.timerCh
		handlerCh = c.handlerCh
		netID = c.netid
		ctx = c.ctx
		askForWorker = true
		logger = c.logger
	case *ClientConn:
		cDone = c.ctx.Done()
		sDone = nil
		timerCh = c.timing.timeOutChan
		handlerCh = c.handlerCh
		netID = c.netid
		ctx = c.ctx
	}

	defer func() {
		if p := recover(); p != nil {
			if logger != nil {
				logger.Errorf("panics: %v\n", p)
			}
		}
		wg.Done()
		if logger != nil {
			logger.Debug("handleLoop go-routine exited")
		}
		c.Close()
	}()

	for {
		select {
		case <-cDone: // connectin closed
			if logger != nil {
				logger.Debug("receiving cancel signal from conn")
			}
			return
		case <-sDone: // server closed
			if logger != nil {
				logger.Debug("receiving cancel signal from server")
			}
			return
		case msgHandler := <-handlerCh:
			msg, handler := msgHandler.message, msgHandler.handler
			if handler != nil {
				if askForWorker {
					WorkerPoolInstance().Put(netID, func() {
						handler(NewContextWithNetID(NewContextWithMessage(ctx, msg), netID), c)
					})
					addTotalHandle()
				} else {
					handler(NewContextWithNetID(NewContextWithMessage(ctx, msg), netID), c)
				}
			}
		case timeout := <-timerCh:
			if timeout != nil {
				timeoutNetID := NetIDFromContext(timeout.Ctx)
				if timeoutNetID != netID {
					if logger != nil {
						logger.Errorf("timeout net %d, conn net %d, mismatched!\n",
							timeoutNetID, netID)
					}
				}
				if askForWorker {
					WorkerPoolInstance().Put(netID, func() {
						timeout.Callback(time.Now(), c.(WriteCloser))
					})
				} else {
					timeout.Callback(time.Now(), c.(WriteCloser))
				}
			}
		}
	}
}
