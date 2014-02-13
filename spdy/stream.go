package spdy

import (
	"bytes"
//	"compress/zlib"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type Stream struct {
	StreamId uint32
	Request  *http.Request
	Response *http.Response
	InFrames []*DataFrame
	handle   Handle
	resw     *io.PipeWriter
	resr     io.Reader
}

func NewStream(streamId uint32) *Stream {
	st := &Stream{
		StreamId: streamId,
		InFrames: make([]*DataFrame, 0, 2),
	}

	return st
}

func (st *Stream) Syn(output chan Frame, req *http.Request, writer io.Writer,
		      zbuf *bytes.Buffer, zwriter io.Writer) {
	st.Request = req

	syn := st.headerToFrame(req)

	if req.Body != nil {
		log.Trace("Stream#%d Request with body", st.StreamId)
		output <- syn
		dat := st.bodyToFrame(req.Body)
		dat.Flags = FLAG_FIN
		output <- dat
	} else {
		log.Trace("Stream#%d Request without body", st.StreamId)
		syn.Flags = FLAG_FIN
		output <- syn
	}
}

func (st *Stream) bodyToFrame(body io.ReadCloser) *DataFrame {
	frame := NewDataFrame(st.StreamId)

	buf := bytes.NewBuffer(make([]byte, 0))

	bs, _ := ioutil.ReadAll(body)
	buf.Write(bs)

	frame.Length = uint32(buf.Len())
	frame.Data = buf

	return frame
}

func (st *Stream) headerToFrame(req *http.Request) *SynStreamFrame {
	frame := NewSynStreamFrame(st.StreamId)

	for k, vs := range req.Header {
		frame.Header[strings.ToLower(k)] = strings.Join(vs, "\x00")
	}

	frame.Header["version"] = req.Proto
	frame.Header["method"] = req.Method
	frame.Header["scheme"] = req.URL.Scheme
	frame.Header["host"] = req.Host

	url := req.URL.Path
	if url == "" {
		url = "/"
	}
	if req.URL.RawQuery != "" {
		url += "?" + req.URL.RawQuery
	}
	if req.URL.Fragment != "" {
		url += "#" + req.URL.Fragment
	}
	frame.Header["url"] = url

	return frame
}

func (st *Stream) ReplyToResponse(srf *SynReplyFrame) {
	header := http.Header{}

	log.Trace("SynReplyFrame header: %v", srf.Header)
	for k, v := range srf.Header {
		vs := strings.Split(v, "\x00")
		for _, s := range vs {
			header.Add(k, s)
		}
	}
	log.Trace("Response header: %v", header)

	st.Response = &http.Response{
		Header:  header,
		Request: st.Request,
	}
	st.Response.Status = header["Status"][0]
	statusCode, _ := strconv.Atoi(st.Response.Status[:3])
	st.Response.StatusCode = statusCode

	st.Response.Proto = header["Version"][0]

	transencoding := header["Content-Encoding"]
	st.Response.TransferEncoding = transencoding

	log.Debug("Stream#%d SynReplyFrame flag %d", st.StreamId, srf.Flags)
	if srf.Flags != FLAG_FIN {
		st.resr, st.resw = io.Pipe()

		st.Response.Body = ioutil.NopCloser(st.resr)
	}

	st.handle(st.StreamId, st.Response, nil)
}

func (st *Stream) DataToResponse(dat *DataFrame) {
	log.Debug("StreamId#%d data to write...", st.StreamId)
	dat.Data.WriteTo(st.resw)

	if dat.Flags == FLAG_FIN {
		st.resw.Close()
	}
}

