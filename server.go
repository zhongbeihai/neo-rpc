package codec

import (
	"encoding/json"
	"log"
	"net"
	"reflect"
	"sync"
)

// ---------- Option ----------
const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType   Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   GobType,
}

// ---------- Server ----------
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
		go server.ServeConn(conn)
	}
}

/*
|   Option{MagicNumber: xxx, CodecType: xxx}   |    Header{ServiceMethod ...} | Body interface{}   |
| <------      fixed JSON coding      ------>  | <-------Coding Method decided by CodeType ------->|
*/
func (server *Server) ServeConn(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	var option Option
	if err := json.NewDecoder(conn).Decode(&option); err != nil {
		log.Println("rpc server: options error", err)
		return
	}
	if option.MagicNumber != MagicNumber {
		log.Println("rpc server: invalid magic number %x", option.MagicNumber)
		return
	}
	f := NewCodeFuncMap[option.CodecType]
	if f == nil {
		log.Println("rpc server: invalid codec type %s", option.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

func (server *Server) serveCodec(cc Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	for {
		request, err := server.readRequest(cc)
		if err != nil {
			if request == nil {
				break
			}
			request.header.Err = err.Error()
			server.sendResponse(cc, &request.header, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go server.handleRequest(cc, request, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

type Request struct {
	header Header
	argv   reflect.Value
	replyv reflect.Value
}

func (server *Server) readRequestHeader(cc Codec) (*Header, error) {

}
func (server *Server) readRequest(cc Codec) (*Request, error) {

}

func (server *Server) sendResponse(cc Codec, h *Header, body interface{}, sending *sync.Mutex) {

}

func (server *Server) handleRequest(cc Codec, req *Request, sending *sync.Mutex, wg *sync.WaitGroup) {

}

func AcceptConn(lis net.Listener) {
	DefaultServer.AcceptConn(lis)
}
