package spdy

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"strings"
)

func ParseHeader(buf *bytes.Buffer) Frame {
	var headFirst uint32
	binary.Read(buf, binary.BigEndian, &headFirst)

	var flagsLength uint32
	binary.Read(buf, binary.BigEndian, &flagsLength)

	frame := NewFrameFromHead(headFirst, flagsLength)

	return frame
}

func (frame *SynReplyFrame) Parse(buf *bytes.Buffer) {
	binary.Read(buf, binary.BigEndian, &frame.StreamId)

	var unused uint16
	binary.Read(buf, binary.BigEndian, &unused)

	zr := wrapBuffer(buf, frame.Length-6)

	frame.Header = readHeader(zr)
}

func (frame *SynStreamFrame) Parse(buf *bytes.Buffer) {
	// =======
}

func readHeader(zr io.Reader) map[string]string {
	var number uint16
	binary.Read(zr, binary.BigEndian, &number)

	header := map[string]string{}

	for i := uint16(0); i < number; i++ {
		var nameLen uint16
		binary.Read(zr, binary.BigEndian, &nameLen)

		nameBytes := make([]byte, nameLen)
		io.ReadFull(zr, nameBytes)
		name := string(nameBytes)
		lowerName := strings.ToLower(name)
		if name != lowerName {
			log.Error("unlowercased header name `%v`", name)
		}
		name = lowerName

		var valueLen uint16
		binary.Read(zr, binary.BigEndian, &valueLen)

		valueBytes := make([]byte, valueLen)
		io.ReadFull(zr, valueBytes)
		values := string(valueBytes)

		log.Debug("- %-20s %s", name+":", values)

		header[name] = values
	}
	return header
}

type BufferWrapper struct {
	lr *io.LimitedReader
	zr io.ReadCloser
}

var wrapper BufferWrapper

func wrapBuffer(buf *bytes.Buffer, length uint32) io.ReadCloser {
	if wrapper.lr == nil {
		wrapper = BufferWrapper{}

		log.Info("init BufferWrapper length=%d", length)
		wrapper.lr = &io.LimitedReader{R: buf, N: int64(length)}

		var err error
		wrapper.zr, err = zlib.NewReaderDict(wrapper.lr, []byte(HeaderDict))
		if err != nil {
			log.Error("%v", err)
		}
	} else {
		wrapper.lr.R = buf
		wrapper.lr.N = int64(length)
	}
	return wrapper.zr
}

func (frame *DataFrame) Parse(buf *bytes.Buffer) {
	log.Trace("Parse Data Frame, length=%d", buf.Len())
	frame.Data = buf
}

func (frame *GoawayFrame) Parse(buf *bytes.Buffer) {
	var lastId uint32
	binary.Read(buf, binary.LittleEndian, &lastId)
}

func (frame *SettingsFrame) Parse(buf *bytes.Buffer) {
	var number uint32
	binary.Read(buf, binary.BigEndian, &number)

	frame.Settings = make([]Setting, 4)

	for i := uint32(0); i < number; i++ {
		var idFlag uint32
		binary.Read(buf, binary.LittleEndian, &idFlag)
		id, flags := idFlag&0x00ffffff, uint8(idFlag>>24)

		var value uint32
		binary.Read(buf, binary.BigEndian, &value)

		frame.Settings[i] = Setting{id, flags, value}
	}

}
