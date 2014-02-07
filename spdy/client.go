package spdy

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
)

type Handle func(uint32, *http.Response, error)

var sessions map[string]Session

func init() {
	sessions = map[string]Session{}
}

func addPort(scheme, host string) string {
	if i := strings.LastIndex(host, ":"); i == -1 {
		switch scheme {
		case "http":
			host += ":80"
		case "https":
			host += ":443"
		default:
			log.Error("unkown scheme: %v", scheme)
		}
	}

	return host
}

func Request(req *http.Request, handle Handle) (uint32, error) {
	host := addPort(req.URL.Scheme, req.Host)

	se, err := getSession(req.URL.Scheme, host)
	if err != nil {
		log.Error("%v", err)
		return 0, err
	}

	id := se.Request(req, handle)
	log.Trace("Wait Response with StreamId %d", id)

	// TODO 当前函数返回后，session 将释放，可能不返回结果
	time.Sleep(50 * time.Millisecond)

	return id, nil
}

func getSession(scheme, host string) (Session, error) {
	se, ok := sessions[host]
	if ok == true {
		if log.DebugEnabled() {
			log.Debug("Use existed session")
		}
	} else {
		var err error
		se, err = initSession(scheme, host)
		if err != nil {
			log.Error("%v", err)
			return nil, err
		}
	}
	return se, nil
}

func initSession(scheme, host string) (s Session, err error) {
	conn, proto, err := connect(scheme, host)
	if err != nil {
		log.Error("%v", err)
		return nil, err
	}

	log.Debug("Connetion %s => %s", conn.LocalAddr(), conn.RemoteAddr())

	switch proto {
	case "http/1.1", "":
		s = NewHttpSession(conn)
	case "spdy/2":
		s = NewSpdySession(conn, conn, 2)
	default:
		log.Fatal("Proto no support")
		err = errors.New("Unreachable code")
	}

	sessions[host] = s
	s.Serve()

	log.Info("Session from %s to %s is Serving", conn.LocalAddr(), conn.RemoteAddr())
	return s, err
}

func connect(scheme, host string) (conn net.Conn, proto string, err error) {
	proto = "spdy/2"
	switch scheme {
	case "http":
		conn, err = DialTCP(host)
	case "https":
		conn, proto, err = DialTLS(host)
	default:
		log.Error("%v", "unreachable code")
		return nil, "", errors.New("Unreachable code")
	}
	if err != nil {
		log.Error("%v", err)
		return nil, "", err
	}

	return conn, proto, nil
}

func DialTCP(host string) (net.Conn, error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func DialTLS(host string) (net.Conn, string, error) {
	config := &tls.Config{
		NextProtos:         []string{"spdy/2", "http/1.1"},
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", host, config)
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
