package spdy

import (
	"bytes"
	"compress/zlib"
	"io"
	"net/http"
)

type Session struct {
	Version uint16

	LastInId  uint32
	LastOutId uint32

	Reader io.Reader

	Writer  io.Writer
	Zbuf    *bytes.Buffer
	Zwriter *zlib.Writer

	Streams map[uint32]*Stream

	Settings []Setting
}

func NewSession(writer io.Writer, reader io.Reader, version uint16) *Session {
	se := &Session{
		Version:   version,
		LastOutId: 0,
		Reader:    reader,
		Writer:    writer,
		Streams:   map[uint32]*Stream{},
	}

	se.Zbuf = new(bytes.Buffer)
	var err error
	se.Zwriter, err = zlib.NewWriterLevelDict(se.Zbuf, zlib.BestCompression, []byte(HeaderDict))
	if err != nil {
		log.Error("%v", err)
		return nil
	}

	return se
}

func (se *Session) Request(req *http.Request) *http.Response {
	streamId := se.nextOutId()

	stream := NewStream(streamId)

	se.Streams[streamId] = stream

	stream.Syn(req)

	for _, frame := range stream.OutFrames {
		syn := frame.(*SynStreamFrame)
		syn.Write(se.Writer, se.Zbuf, se.Zwriter)
	}

	return stream.WaitResponse()
}

func (se *Session) nextOutId() uint32 {
	if se.LastOutId == 0 {
		se.LastOutId = 1
	} else {
		se.LastOutId += 2
	}

	return se.LastOutId
}

func (se *Session) Serve() {
	go se.receive()
}

func (se *Session) receive() {
	for {
		head := make([]byte, 8)
		io.ReadFull(se.Reader, head)
		hBuf := bytes.NewBuffer(head)

		log.Debug("Receive frame head: %x", head)

		frame := ParseHeader(hBuf)
		log.Debug("Receive frame head: %v", frame.Head())

		body := make([]byte, frame.Len())
		io.ReadFull(se.Reader, body)
		bBuf := bytes.NewBuffer(body)

		log.Debug("Receive frame body: %x", body)

		go se.parseBody(frame, bBuf)
	}
}

func (se *Session) parseBody(frame Frame, bbuf *bytes.Buffer) {
	frame.Parse(bbuf)

	switch frame.(type) {
	case *SynReplyFrame:
		reply, _ := frame.(*SynReplyFrame)
		se.Streams[reply.StreamId].ReplyToResponse(reply)
	case *DataFrame:
		dat, _ := frame.(*DataFrame)
		se.Streams[dat.StreamId].DataToResponse(dat)
	case *SynStreamFrame:
		log.Fatal("%v", "unimplemented NewCtrlFrame SynStreamFrame")
	case *RstStreamFrame:
		log.Fatal("%v", "unimplemented NewCtrlFrame RstStreamFrame")
	case *SettingsFrame:
		set, _ := frame.(*SettingsFrame)
		se.settings(set)
	case *NoopFrame:
		log.Fatal("%v", "unimplemented NewCtrlFrame NoopFrame")
	case *PingFrame:
		log.Fatal("%v", "unimplemented NewCtrlFrame PingFrame")
	case *GoawayFrame:
		log.Fatal("%v", "unimplemented NewCtrlFrame GoawayFrame")
	case *HeadersFrame:
		log.Fatal("%v", "unimplemented NewCtrlFrame HeadersFrame")
	default:
		log.Error("%v", "unreachable code")
	}
}

func (se *Session) settings(set *SettingsFrame) {
	se.Settings = set.Settings
}
