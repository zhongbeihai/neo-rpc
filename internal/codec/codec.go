package codec

import "io"

/*
Standart RPC call: err = client.Call("Arithy.Multiply", args, &reply)
Abstract parameters and return values in the **body**
Abstract other value in the **Header**
*/
type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number of request, chosen by clients
	Err           string
}

// Abstract encoding and decoding function
type Codec interface {
	io.Closer
	ReadHeader(h *Header) error
	ReadBody(body interface{}) error
	Write(h *Header, body interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

// mapping Type to NewCodecFunc
var NewCodeFuncMap map[Type]NewCodecFunc

func init() {
	NewCodeFuncMap = make(map[Type]NewCodecFunc)
	NewCodeFuncMap[GobType] = NewGobCodec
}
