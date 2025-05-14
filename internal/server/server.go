package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"neo-rpc/internal/codec"
	"net"
	"reflect"
	"sync"
)

// -------------------- Option --------------------
const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType   codec.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

/*
client → (conn) → HandleConn
					├─ Accept Conn
					├─ Handle conn, read Option (codec)
					└─ serveCodec  (decode request and encode response)
						└─ for each request:
							├─ readRequestHeader
							├─ readRequest (decode argv)
							├─ handleRequest (log + construct reply)
							└─ sendResponse

*/
// -------------------- Server --------------------
type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (server *Server) AcceptConn(listen net.Listener) {
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("rpc server: accept connection error:", err)
			return
		}
		go server.HandleConn(conn)
	}
}

/*
|   Option{MagicNumber: xxx, CodecType: xxx}   |    Header{ServiceMethod ...} | Body interface{}   |
| <------      fixed JSON coding      ------>  | <-------Coding Method decided by CodeType ------->|
*/
func (server *Server) HandleConn(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	// Read Option (communication protocol negotiation) and decide what encoding method to use
	var option Option
	if err := json.NewDecoder(conn).Decode(&option); err != nil {
		log.Println("rpc server: options error", err)
		return
	}
	if option.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", option.MagicNumber)
		return
	}

	// Get the corresponding encoding and decoding function
	f := codec.NewCodeFuncMap[option.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", option.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	// Enter the main loop: continuously read requests and respond (using the specified encoder)
	for {
		request, err := server.readRequest(cc)
		if err != nil {
			if request == nil {
				break
			}
			request.header.Err = err.Error()
			server.sendResponse(cc, request.header, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go server.handleRequest(cc, request, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

type Request struct {
	header *codec.Header
	argv   reflect.Value
	replyv reflect.Value
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*Request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}

	req := &Request{
		header: h,
	}
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}

	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()

	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *Request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	log.Println(req.header, " ", req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("neorpc res %d", req.header.Seq))
	server.sendResponse(cc, req.header, req.replyv.Interface(), sending)
}

func AcceptConn(lis net.Listener) {
	DefaultServer.AcceptConn(lis)
}
