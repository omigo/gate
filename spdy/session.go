package spdy

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
)

const FRAME_BUFFER_SIZE = 100

type Session struct {
	Version   uint16
	output    chan Frame
	input     chan Frame
	LastInId  uint32
	LastOutId uint32
	r         net.Conn
	lr        *io.LimitedReader
	zr        io.ReadCloser
	w         io.Writer
	buf       *bytes.Buffer
	zw        *zlib.Writer
	Streams   map[uint32]*Stream
	Settings  []Setting
}

func NewSession(writer io.Writer, reader net.Conn, version uint16) *Session {
	se := &Session{
		Version:   version,
		output:    make(chan Frame, FRAME_BUFFER_SIZE),
		input:     make(chan Frame, FRAME_BUFFER_SIZE),
		LastOutId: 0,
		r:         reader,
		w:         writer,
		Streams:   map[uint32]*Stream{},
	}

	se.buf = new(bytes.Buffer)
	var err error
	se.zw, err = zlib.NewWriterLevelDict(se.buf, zlib.BestCompression, []byte(HeaderDict))
	if err != nil {
		log.Error("%v", err)
		return nil
	}

	return se
}

func (se *Session) Serve() {
	go se.send()
	//	go se.receiveDebug()
	go se.receive()
	go se.toResponse()
}

func (se *Session) send() {
	for frame := range se.output {
		switch frame.(type) {
		case *SynStreamFrame:
			syn, _ := frame.(*SynStreamFrame)
			syn.write(se.w, se.buf, se.zw)
		case *DataFrame:
			dat, _ := frame.(*DataFrame)
			dat.Write(se.w)
		case *SynReplyFrame:
			log.Fatal("%v", "unimplemented NewCtrlFrame SynStreamFrame")
		case *RstStreamFrame:
			log.Fatal("%v", "unimplemented NewCtrlFrame RstStreamFrame")
		case *SettingsFrame:
			log.Fatal("%v", "unimplemented NewCtrlFrame SettingsFrame")
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
	log.Info("Session output frame Closed")
}

func (se *Session) receiveDebug() {
	for i := 1; i < 100; i++ {
		var headFirst uint32
		binary.Read(se.r, binary.BigEndian, &headFirst)
		var flagsLength uint32
		binary.Read(se.r, binary.BigEndian, &flagsLength)

		log.Debug("Receive head from Session: %08x %08x", headFirst, flagsLength)

		blen := flagsLength & 0x00ffffff
		body := make([]byte, blen)
		se.r.Read(body)

		log.Debug("Receive body from Session: (%d)%x", blen, body)
	}
}

func (se *Session) receive() {
	for {
		var headFirst uint32
		binary.Read(se.r, binary.BigEndian, &headFirst)

		log.Debug("Receive head from Session: %08x", headFirst)

		var frame Frame
		var err error
		if headFirst&0x80000000 != 0 {
			frame, err = se.readCtrlFrame(headFirst)
		} else {
			frame, err = se.readDataFrame(headFirst)
		}
		if err != nil {
			log.Error("%v", err)
			break
		}

		log.Debug("Frame to input queue")
		se.input <- frame
	}
}

func (se *Session) readDataFrame(headFirst uint32) (Frame, error) {
	log.Debug("Read DataFrame with headFirst %08x", headFirst)

	streamId := headFirst & 0x7fffffff

	if _, ok := se.Streams[streamId]; !ok {
		return nil, errors.New("DataFrame streamId not exist in session")
	}

	var flagsLength uint32
	binary.Read(se.r, binary.BigEndian, &flagsLength)

	f := &DataFrame{
		StreamId: streamId,
		Flags:    uint8(flagsLength >> 24),
		Length:   flagsLength & 0x00ffffff,
	}

	if f.StreamId == 0 {
		return nil, errors.New("DataFrame StreamId must not 0")
	}

	return f.ReadBody(se.r)
}

func (se *Session) readCtrlFrame(headFirst uint32) (Frame, error) {
	log.Debug("Read CtrlFrame with headFirst %08x", headFirst)
	var flagsLength uint32
	binary.Read(se.r, binary.BigEndian, &flagsLength)

	head := CtrlFrameHead{
		Version: uint16(headFirst & 0x7fff0000 >> 16),
		Type:    uint16(headFirst & 0xffff),
		Flags:   uint8(flagsLength >> 24),
		Length:  flagsLength & 0xffffff,
	}

	if head.Version == 0 {
		return nil, errors.New("CtrlFrame Version must not 0")
	} else if head.Length == 0 {
		return nil, errors.New("CtrlFrame Length must not 0")
	} else if head.Type == 0 {
		return nil, errors.New("CtrlFrame Type must not 0")
	}

	switch head.Type {
	case SYN_REPLY:
		log.Debug("read SYN_REPLY")
		reply := &SynReplyFrame{CtrlFrameHead: head}

		reply.Read(se.r)
		// read header
		se.wrapReader(reply.Length - 6)
		reply.ReadHeader(se.zr)

		return reply, nil
	case SETTINGS:
		set := &SettingsFrame{CtrlFrameHead: head}
		set.Read(se.r)

		return set, nil
	case SYN_STREAM:
		return nil, errors.New("unimplemented SYN_STREAM")
	case GOAWAY:
		ga := &GoawayFrame{CtrlFrameHead: head}
		ga.Read(se.r)

		return ga, nil
	case RST_STREAM:
		return nil, errors.New("unimplemented NewCtrlFrame RST_STREAM")
	case NOOP:
		return nil, errors.New("unimplemented NewCtrlFrame NOOP")
	case PING:
		return nil, errors.New("unimplemented NewCtrlFrame PING")
	case HEADERS:
		return nil, errors.New("unimplemented NewCtrlFrame HEADERS")
	default:
		return nil, errors.New("unknown CtrlFrame Type")

	}
	return nil, errors.New("unreadable code")
}

func (se *Session) wrapReader(length uint32) {
	if se.lr == nil {
		log.Info("init BufferWrapper length=%d", length)
		se.lr = &io.LimitedReader{R: se.r, N: int64(length)}

		var err error
		se.zr, err = zlib.NewReaderDict(se.lr, []byte(HeaderDict))
		if err != nil {
			log.Error("%v", err)
		}
	} else {
		log.Info("Chang LimitedReader length to %d", length)
		se.lr.N = int64(length)
	}
}

func (se *Session) toResponse() {
	for frame := range se.input {
		switch frame.(type) {
		case *SynReplyFrame:
			log.Debug("SynReplyFrame from input queue")
			reply, _ := frame.(*SynReplyFrame)
			se.Streams[reply.StreamId].ReplyToResponse(reply)
		case *DataFrame:
			log.Debug("DataFrame from input queue")
			dat, _ := frame.(*DataFrame)
			if st, ok := se.Streams[dat.StreamId]; ok {
				log.Debug("Stream#%d exist in Session", dat.StreamId)
				st.DataToResponse(dat)
			} else {
				log.Error("Stream#%d not exist in Session", dat.StreamId)
				continue
			}
		case *SynStreamFrame:
			log.Fatal("%v", "unimplemented NewCtrlFrame SynStreamFrame")
		case *RstStreamFrame:
			log.Fatal("%v", "unimplemented NewCtrlFrame RstStreamFrame")
		case *SettingsFrame:
			log.Debug("SettingsFrame from input queue")
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
	log.Info("Session output frame Closed")
}

func (se *Session) Request(req *http.Request) uint32 {
	log.Debug("Request %s", req.URL.String())

	streamId := se.nextOutId()
	stream := NewStream(streamId)
	se.Streams[streamId] = stream

	go stream.Syn(se.output, req, se.w, se.buf, se.zw)

	return streamId
}

func (se *Session) Response(streamId uint32) *http.Response {
	defer delete(se.Streams, streamId)

	return se.Streams[streamId].WaitResponse()
}

func (se *Session) nextOutId() uint32 {
	if se.LastOutId == 0 {
		se.LastOutId = 1
	} else {
		se.LastOutId += 2
	}

	return se.LastOutId
}

func (se *Session) settings(set *SettingsFrame) {
	se.Settings = set.Settings
}
