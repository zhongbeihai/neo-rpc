package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// implement Codec interface
type GobCodec struct {
	conn    io.ReadWriteCloser
	buf     *bufio.Writer
	decoder *gob.Decoder
	encoder *gob.Encoder
}

// Close implements Codec.
func (g *GobCodec) Close() error {
	return g.conn.Close()
}

// ReadBody implements Codec.
func (g *GobCodec) ReadBody(body interface{}) error {
	return g.decoder.Decode(body)
}

// ReadHeader implements Codec.
func (g *GobCodec) ReadHeader(h *Header) error {
	return g.decoder.Decode(h)
}

// Write implements Codec.
func (g *GobCodec) Write(h *Header, body interface{}) error {
	defer func ()  {
		err := g.buf.Flush()
		if err != nil{
			_ = g.Close()
		}
	}()

	if err := g.encoder.Encode(h); err != nil{
		log.Println("rpc codec: gob error in encoding header", err)
		return err
	}
	if err := g.encoder.Encode(body); err != nil{
		log.Println("rpc codec: gob error in encoding body", err)
	}
	return nil
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn:    conn,
		buf:     buf,
		decoder: gob.NewDecoder(conn),
		encoder: gob.NewEncoder(buf),
	}
}
