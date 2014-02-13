package spdy

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
)

func (f *SynStreamFrame) write(w io.Writer, buf *bytes.Buffer, zw *zlib.Writer) {

	zheader := writeHeader(f.Header, buf, zw)

	blen := len(zheader) + 18
	log.Trace("New Buffer with bytes size = %d", blen)
	bs := make([]byte, 0, blen)
	b := bytes.NewBuffer(bs)

	b.Write([]byte{0x80, 0x02, 0x00, 0x01})

	flagsLength := uint32(f.Flags<<24) + uint32(len(zheader)) + 10
	b.Write(uint32ToBytes(flagsLength))
	b.Write(uint32ToBytes(f.StreamId))
	b.Write(uint32ToBytes(f.AssociatedId))

	priority := f.Priority << 14
	b.Write(uint16ToBytes(priority))

	b.Write(zheader)

	if log.TraceEnabled() {
		log.Trace("zlib header: %x", zheader)
		log.Trace("Write to Session: (%d)%x", b.Len(), b.Bytes())
	}
	b.WriteTo(w)

	log.Debug("Send frame: %v", f)
}

func uint32ToBytes(u uint32) []byte {
	bs := []byte{
		byte(u >> 24),
		byte(u >> 16),
		byte(u >> 8),
		byte(u),
	}

	return bs
}
func uint16ToBytes(u uint16) []byte {
	bs := []byte{
		byte(u >> 8),
		byte(u),
	}

	return bs
}

func writeHeader(header map[string]string, buf *bytes.Buffer, zw *zlib.Writer) []byte {
	defer buf.Reset()

	binary.Write(zw, binary.BigEndian, uint16(len(header)))

	for k, v := range header {
		binary.Write(zw, binary.BigEndian, uint16(len(k)))
		io.WriteString(zw, k)
		binary.Write(zw, binary.BigEndian, uint16(len(v)))
		io.WriteString(zw, v)
	}
	zw.Flush()

	return buf.Bytes()
}

func (f *DataFrame) write(w io.Writer) {
	blen := f.Length + 8
	log.Trace("New Buffer with bytes size = %d", blen)
	bs := make([]byte, blen)
	b := bytes.NewBuffer(bs)
	b.Reset()

	binary.Write(b, binary.BigEndian, uint32ToBytes(f.StreamId))
	flagsLength := (uint32(f.Flags) << 24) + uint32(f.Length)
	binary.Write(b, binary.BigEndian, uint32ToBytes(flagsLength))

	f.Data.WriteTo(b)

	b.WriteTo(w)

	log.Debug("Send DataFrame: %v", f)
}
