package codec

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

type JsonCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *json.Decoder
	enc  *json.Encoder
}

var _ Codec = (*JsonCodec)(nil)

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &JsonCodec{
		conn: conn,
		buf:  buf,
		dec:  json.NewDecoder(conn),
		enc:  json.NewEncoder(buf),
	}
}

func (c *JsonCodec) ReadHeader(h *Header) error {
	return nil
}

func (c *JsonCodec) ReadBody(body interface{}) error {
	return nil
}

func (c *JsonCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()

	if err := c.dec.Decode(h); err != nil {
		log.Println("rpc codec : json error encoding header:", err)
		return err
	}

	if err := c.dec.Decode(body); err != nil {
		log.Println("rpc codec : json error encoding header:", err)
		return err
	}

	return nil
}

func (c *JsonCodec) Close() error {
	return c.conn.Close()
}
