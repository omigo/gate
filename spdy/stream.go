package spdy

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
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
}

func NewStream(streamId uint32) *Stream {
	st := &Stream{
		StreamId: streamId,
		InFrames: make([]*DataFrame, 0, 2),
	}

	return st
}

func (st *Stream) Syn(output chan Frame, req *http.Request, writer io.Writer,
	zbuf *bytes.Buffer, zwriter *zlib.Writer) {
	st.Request = req

	frame := st.headerToFrame(req)
	output <- frame

	if req.Body != nil {
		// TODO
		// frame = st.headerToFrame(req.Body)
		// output <- frame
	}

	frame.Flags = FLAG_FIN
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

	st.Response.TransferEncoding = header["Content-Encoding"]

	log.Debug("Stream#%d SynReplyFrame flag %d", st.StreamId, srf.Flags)
	if srf.Flags == FLAG_FIN {
		st.endResponse()
	}
}

func (st *Stream) DataToResponse(dat *DataFrame) {
	st.InFrames = append(st.InFrames, dat)
	log.Debug("Stream#%d InFrames len=%d after append DateFrame", st.StreamId, len(st.InFrames))

	log.Debug("Stream#%d DataFrame flag %d", st.StreamId, dat.Flags)
	if dat.Flags == FLAG_FIN {
		st.endDataFrame()
	}
}

func (st *Stream) endDataFrame() {
	var mr io.Reader
	res := st.Response
	if res == nil {
		log.Error("Stream#%d DateFrame must after SynReplyFrame", st.StreamId)
		return
	} else if len(res.TransferEncoding) > 0 && res.TransferEncoding[0] == "gzip" {
		for _, df := range st.InFrames {
			r, _ := gzip.NewReader(df.Data)
			if mr == nil {
				mr = io.MultiReader(r)
			} else {
				mr = io.MultiReader(mr, r)
			}
		}
	} else {
		for _, df := range st.InFrames {
			if mr == nil {
				mr = io.MultiReader(df.Data)
			} else {
				mr = io.MultiReader(mr, df.Data)
			}
		}
	}
	res.Body = ioutil.NopCloser(mr)

	st.endResponse()
}

func (st *Stream) endResponse() {
	log.Info("Stream#%d Response is ok, stream end", st.StreamId)
	st.handle(st.StreamId, st.Response, nil)
}
