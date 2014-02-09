package spdy

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
)

const FRAME_BUFFER_SIZE = 100

type Session interface {
	Serve()
	Request(*http.Request, Handle) uint32
}

type HttpSession struct {
	client *httputil.ClientConn
}

func NewHttpSession(conn net.Conn) *HttpSession {
	hs := &HttpSession{
		client: httputil.NewClientConn(conn, nil),
	}
	return hs
}

func (hs *HttpSession) Serve() {
	// do nothing
}

func (hs *HttpSession) Request(req *http.Request, handle Handle) uint32 {
	res, err := hs.client.Do(req)
	if err != nil {
		log.Error("%v", err)
		return 0
	}

	// callback
	handle(0, res, nil)

	return 0
}

type SpdySession struct {
	Version   uint16
	output    chan Frame
	input     chan Frame
	LastInId  uint32
	LastOutId uint32
	r         io.Reader
	lr        *io.LimitedReader
	zr        io.ReadCloser
	w         io.Writer
	buf       *bytes.Buffer
	zw        *zlib.Writer
	Streams   map[uint32]*Stream
	Settings  []Setting
}

func NewSpdySession(writer io.Writer, reader io.Reader, version uint16) Session {
	se := &SpdySession{
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

func (se *SpdySession) Request(req *http.Request, handle Handle) uint32 {
	log.Debug("Request %s", req.URL.String())

	streamId := se.nextOutId()

	go func() {
		stream := NewStream(streamId)
		stream.handle = handle
		se.Streams[streamId] = stream

		stream.Syn(se.output, req, se.w, se.buf, se.zw)
	}()
	return streamId
}

func (se *SpdySession) nextOutId() uint32 {
	if se.LastOutId == 0 {
		se.LastOutId = 1
	} else {
		se.LastOutId += 2
	}

	return se.LastOutId
}

func (se *SpdySession) Serve() {
	go se.receive()
	go se.send()
	go se.toResponse()

	log.Info("Session is serving")
}


func (se *SpdySession) send() {
	for frame := range se.output {
		switch frame.(type) {
		case *SynStreamFrame:
			syn, _ := frame.(*SynStreamFrame)
			syn.write(se.w, se.buf, se.zw)
		case *DataFrame:
			dat, _ := frame.(*DataFrame)
			dat.write(se.w)
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

func (se *SpdySession) receive() {
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

func (se *SpdySession) readDataFrame(headFirst uint32) (Frame, error) {
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

func (se *SpdySession) readCtrlFrame(headFirst uint32) (Frame, error) {
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

func (se *SpdySession) wrapReader(length uint32) {
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

func (se *SpdySession) toResponse() {
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
}

func (se *SpdySession) settings(set *SettingsFrame) {
	se.Settings = set.Settings
}
