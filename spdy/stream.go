package spdy

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type Stream struct {
	StreamId  uint32
	OutFrames []Frame
	InFrames  []Frame

	Request  *http.Request
	Response *http.Response

	Fin chan bool
}

func (st *Stream) WaitResponse() *http.Response {
	log.Info("Stream#%d Reply wait", st.StreamId)
	<-st.Fin
	log.Info("Stream#%d Reply FIN", st.StreamId)
	return st.Response
}

func NewStream(streamId uint32) *Stream {
	st := &Stream{
		StreamId:  streamId,
		OutFrames: make([]Frame, 0, 2),
		InFrames:  make([]Frame, 0, 2),
		Fin: make(chan bool),}

	return st
}

func (st *Stream) Syn(req *http.Request) {
	st.Request = req

	frame := st.reqToFrames(req)

	// TODO must use method not property
	frame.Flags = FLAG_FIN
	st.OutFrames = append(st.OutFrames, frame)

	log.Trace("Stream#%d OutFrames len = %d", st.StreamId, len(st.OutFrames))
}

func (st *Stream) reqToFrames(req *http.Request) *SynStreamFrame {
	frame := NewSynStreamFrame(st.StreamId)

	for k, vs := range req.Header {
		frame.Header[strings.ToLower(k)] = strings.Join(vs, "\x00")
	}

	frame.Header["method"] = req.Method
	frame.Header["scheme"] = req.URL.Scheme
	frame.Header["host"] = req.Host
	// TODO args and fragments ...
	frame.Header["url"] = req.URL.Path
	frame.Header["version"] = req.Proto

	return frame
}

func (st *Stream) ReplyToResponse(srf *SynReplyFrame) {
	st.InFrames = append(st.InFrames, srf)
	log.Debug("Stream#%d InFrames len=%d after append SynReplyFrame", st.StreamId, len(st.InFrames))

	header := http.Header{}

	for k, v := range srf.Header {
		vs := strings.Split(v, "\x00")
		for _, s := range vs {
			header.Add(k, s)
		}
	}
	log.Trace("%v", header)

	st.Response = &http.Response{
		Header: header,
	}
	st.Response.Status = header["Status"][0]
	statusCode, _ := strconv.Atoi(st.Response.Status[:3])
	st.Response.StatusCode = statusCode

	log.Trace("Version=%v", header["Version"][0])
	st.Response.Proto = header["Version"][0]

	st.Response.TransferEncoding = header["Content-Encoding"]
}

func (st *Stream) DataToResponse(dat *DataFrame) {
	st.InFrames = append(st.InFrames, dat)
	log.Debug("Stream#%d InFrames len=%d after append DateFrame", st.StreamId, len(st.InFrames))

	log.Debug("Stream#%d this DataFrame flag %d", st.StreamId, dat.Flags)
	if dat.Flags == FLAG_FIN {
		// TODO how to Response multi DateFrame ?
		if st.Response.TransferEncoding[0] == "gzip" {
			st.Response.Body, _ = gzip.NewReader(dat.Data)
		} else {
			st.Response.Body = ioutil.NopCloser(dat.Data)
		}
		log.Info("Stream#%d Response is ok, stream end", st.StreamId)
		st.Fin <- true
	}
}
