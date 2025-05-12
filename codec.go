package codec

import "io"

type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number of request, chosen by clients
	Err           string
}

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

var NewCodeFuncMap map[Type]NewCodecFunc

func init() {
	NewCodeFuncMap = make(map[Type]NewCodecFunc)
	NewCodeFuncMap[GobType] = NewGobCodec
}
