package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"neo-rpc/internal/codec"
	"neo-rpc/internal/server"
	"net"
	"sync"
)

// func (t *T) MethodName(argType T1, replyType *T2) error

/*
Call represent an active RPC
*/
type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	cc      codec.Codec
	opt     *server.Option
	sending sync.Mutex // protect following,
	// ensure request sending in order, not mixing together
	header   codec.Header
	mu       sync.Mutex // protect following
	seq      uint64     // start from **number > 0**
	pending  map[uint64]*Call
	Closing  bool // client has called Close()
	Shutdown bool // server has told client to stop
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shutting down")

// ---------- process status of client ----------
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.Closing {
		return ErrShutdown
	}
	client.Closing = true
	return client.cc.Close()
}

func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()

	return !client.Shutdown && !client.Closing
}

// ---------- process calls ----------
func (client *Client) RegisterCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.Closing || client.Shutdown {
		return 0, ErrShutdown
	}

	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

func (client *Client) removeCall(sequence uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()

	call := client.pending[sequence]
	delete(client.pending, sequence)
	return call
}

func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.Shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

// ---------- receive call ----------
func (client *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		case call == nil:
			err = client.cc.ReadBody(nil)
		case h.Err != "":
			call.Error = fmt.Errorf(h.Err)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	client.terminateCalls(err)
}

// ---------- Construct Client ----------
func NewClient(conn net.Conn, opt *server.Option) (*Client, error) {
	f := codec.NewCodeFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: Codec type error", err)
		return nil, err
	}

	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error:", err)
		_ = conn.Close()
		return nil, err
	}

	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *server.Option) *Client {
	client := &Client{
		seq:     1,
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

func parseOptions(opts ...*server.Option) (*server.Option, error){
	if len(opts) == 0 || opts[0] == nil{
		return server.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}

	opt := opts[0]
	opt.MagicNumber = server.DefaultOption.MagicNumber
	if opt.CodecType == ""{
		opt.CodecType = server.DefaultOption.CodecType
	}

	return opt, nil
}

func Dial(network, address string, opts ...*server.Option) (client *Client, err error){
	opt, err := parseOptions(opts...)
	if  err != nil {
		return nil, err
	}

	conn, err := net.Dial(network, address)
	if err != nil{
		return nil, err
	}

	defer func ()  {
		if client == nil{
			_ = conn.Close()
		}
	}()

	return NewClient(conn, opt)
}

func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	// register this call
	seq, err := client.RegisterCall(call)
	if  err != nil {
		call.Error = err
		call.done()
		return
	}

	// prepare request header
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Err = ""

	if err := client.cc.Write(&client.header, call.Args); err != nil{
		call := client.removeCall(seq)
		if call != nil{
			call.Error = err
			call.done()
		}
	}
}


func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call{
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)
	return call
}