package client

import (
	"errors"
	"io"
	"neo-rpc/internal/codec"
	"neo-rpc/internal/server"
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
	cc       codec.Codec
	opt      *server.Option
	sending  sync.Mutex // protect following, 
	// ensure request sending in order, not mixing together
	header   codec.Header
	mu       sync.Mutex // protect following
	seq      uint64 // start from **number > 0**
	pending  map[uint64]*Call
	Closing  bool // client has called Close()
	Shutdown bool // server has told client to stop
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shutting down")

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

func (client *Client) RegisterCall(call *Call) (uint64, error){
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

func (client *Client) removeCall(sequence uint64) *Call{
	client.mu.Lock()
	defer client.mu.Unlock()

	call := client.pending[sequence]
	delete(client.pending, sequence)
	return call
}

func (client *Client) terminateCalls(err error){
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.Shutdown = true
	for _, call := range client.pending{
		call.Error = err
		call.done()
	}
}

