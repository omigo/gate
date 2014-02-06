package spdy

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

type Handle func(uint32, *http.Response, error)

var sessions map[string]*Session

func init() {
	sessions = map[string]*Session{}
}

func Request(req *http.Request, handle Handle) (uint32, error) {
	var host = req.Host

	if i := strings.LastIndex(host, ":"); i == -1 {
		switch req.URL.Scheme {
		case "http":
			host += ":80"
		case "https":
			host += ":443"
		default:
			log.Error("unkown scheme: %v", req.URL.Scheme)
			return 0, errors.New("Unreachable code")
		}
	}

	if se, ok := sessions[host]; ok == true {
		if log.DebugEnabled() {
			conn := net.Conn(se.r)
			log.Debug("Use existed session %v => %v",
				conn.LocalAddr(), conn.RemoteAddr())
		}
		id := se.Request(req, handle)
		log.Trace("Wait Response with StreamId %d", id)

		return id, errors.New("Unreachable code")
	}

	var conn net.Conn
	proto := "spdy/2"
	var err error
	switch req.URL.Scheme {
	case "http":
		conn, err = DialTCP(host)
	case "https":
		conn, proto, err = DialTLS(host)
	default:
		log.Error("%v", "unreachable code")
		return 0, errors.New("Unreachable code")
	}
	if err != nil {
		log.Error("%v", err)
		return 0, err
	}
	defer conn.Close()

	switch proto {
	case "http/1.1", "":
		client := httputil.NewClientConn(conn, nil)
		res, _ := client.Do(req)

		// callback
		handle(0, res, nil)

		return 0, nil
	case "spdy/2":
		session := NewSession(conn, conn, 2)
		sessions[host] = session

		log.Info("New Session %s => %s", conn.LocalAddr(), conn.RemoteAddr())

		session.Serve()

		id := session.Request(req, handle)
		log.Trace("Wait Response with StreamId %d", id)

		return id, nil
	default:
		log.Fatal("Proto no support")
	}
	return 0, errors.New("Unreachable code")
}

func DialTCP(host string) (net.Conn, error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func DialTLS(host string) (net.Conn, string, error) {
	config := tls.Config{
		NextProtos:         []string{"spdy/2", "http/1.1"},
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", host, &config)
	if err != nil {
		log.Error("%v", err)
		return nil, "", err
	}

	state := conn.ConnectionState()
	if log.Level <= INFO {
		for _, v := range state.PeerCertificates {
			publicKey, err := x509.MarshalPKIXPublicKey(v.PublicKey)
			if err != nil {
				log.Error("%v", err)
				return nil, "", err
			}
			log.Info("PublicKey = %x\n", publicKey)
			log.Info("Subject = %v", v.Subject)
		}
	}

	proto := state.NegotiatedProtocol

	return conn, proto, nil
}
