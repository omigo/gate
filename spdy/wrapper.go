package spdy

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
)

func (frame *SynStreamFrame) Write(writer io.Writer, zbuf *bytes.Buffer, zwriter *zlib.Writer) {
	binary.Write(writer, binary.BigEndian, uint32(0x80020001))

	writeHeader(frame.Header, zwriter)
	zheader := zbuf.Bytes()

	binary.Write(writer, binary.BigEndian, uint32(frame.Flags<<24)+uint32(len(zheader))+10)
	binary.Write(writer, binary.BigEndian, frame.StreamId)
	binary.Write(writer, binary.BigEndian, frame.AssociatedId)
	binary.Write(writer, binary.BigEndian, frame.Priority<<14)

	writer.Write(zheader)
}

func writeHeader(header map[string]string, zwriter *zlib.Writer) {
	binary.Write(zwriter, binary.BigEndian, uint16(len(header)))

	for k, v := range header {
		binary.Write(zwriter, binary.BigEndian, uint16(len(k)))
		io.WriteString(zwriter, k)
		binary.Write(zwriter, binary.BigEndian, uint16(len(v)))
		io.WriteString(zwriter, v)
	}
	zwriter.Flush()
}
