package tunnel

import (
	"encoding/binary"
	"github.com/rueian/aerial/pkg/buffer"
	"io"
)

type Message struct {
	Type byte
	Conn uint32
	Body []byte
}

func (m *Message) ReadFrom(r io.Reader) (int64, error) {
	buf := buffer.Pool5.Get()

	if n, err := io.ReadFull(r, buf); err != nil {
		buffer.Pool5.Put(buf)
		return int64(n), err
	}

	m.Type = buf[0]
	body := make([]byte, binary.BigEndian.Uint32(buf[1:5]))
	buffer.Pool5.Put(buf)
	n, err := io.ReadFull(r, body)

	if n >= 4 {
		m.Conn = binary.BigEndian.Uint32(body[:4])
		m.Body = body[4:]
	}

	return int64(n + 5), err
}

func (m *Message) WriteTo(w io.Writer) (int64, error) {
	buf := buffer.Pool9.Get()
	buf[0] = m.Type
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(m.Body)+4))
	binary.BigEndian.PutUint32(buf[5:9], m.Conn)

	if n, err := w.Write(buf); err != nil {
		buffer.Pool9.Put(buf)
		return int64(n), err
	}
	buffer.Pool9.Put(buf)

	n, err := w.Write(m.Body)
	return int64(n + 9), err
}
