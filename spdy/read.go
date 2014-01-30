package spdy

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
)

func (frame *SynReplyFrame) Read(r io.Reader) {
	binary.Read(r, binary.BigEndian, &frame.StreamId)

	var unused uint16
	binary.Read(r, binary.BigEndian, &unused)
}

func (frame *SynReplyFrame) ReadHeader(zr io.Reader) {
	var number uint16
	binary.Read(zr, binary.BigEndian, &number)
	log.Debug("SynReplyFrame header number %d", number)

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

		log.Debug("%-20s %s", name+":", values)

		header[name] = values
	}
	frame.Header = header
}

func (frame *DataFrame) ReadBody(r io.Reader) (Frame, error) {
	body := make([]byte, frame.Len())
	io.ReadFull(r, body)
	frame.Data = bytes.NewBuffer(body)

	log.Trace("Parse Data Frame, length=%d", frame.Len())

	return frame, nil
}

func (frame *GoawayFrame) Read(r io.Reader) {
	var lastId uint32
	binary.Read(r, binary.LittleEndian, &lastId)

	log.Debug("Receive GoawayFrame with last good stream id %d", lastId)
}

func (frame *SettingsFrame) Read(r io.Reader) {
	var number uint32
	binary.Read(r, binary.BigEndian, &number)

	frame.Settings = make([]Setting, 4)

	for i := uint32(0); i < number; i++ {
		var idFlag uint32
		binary.Read(r, binary.LittleEndian, &idFlag)
		id, flags := idFlag&0x00ffffff, uint8(idFlag>>24)

		var value uint32
		binary.Read(r, binary.BigEndian, &value)

		frame.Settings[i] = Setting{id, flags, value}
	}

}
