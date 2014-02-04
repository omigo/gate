package main

import (
	"./spdy"
	"bufio"
	"io"
	"net/http"
)

func main() {
	//	rawurl := "https://www.google.com"
	//	rawurl := "https://wordpress.com"
	rawurl := "http://127.0.0.1:2800/"
	//	rawurl := "https://isspdyenabled.com/"
	verbose := "vvv"

	level := byte(3 - len(verbose))
	log := spdy.GetLogger()
	log.SetLevel(level)

	req, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		log.Error("%v", err)
	}

	req.Header.Set("Cache-Control", "nostore")
	req.Header.Set("accept-encoding", "gzip, deflate")
	req.Header.Set("user-agent", "gate/0.0.1")

	times := 2
	end := make(chan bool, times)
	for i := 0; i < times; i++ {
		go request(req, log, end)
	}

	for i := 0; i < times; i++ {
		<-end
	}
}

func request(req *http.Request, log *spdy.Logger, end chan<- bool) {
	res, err := spdy.Request(req)
	if err != nil {
		log.Error("< %v", err)
	}

	log.Debug("%v", res.Header)
	rd := bufio.NewReader(res.Body)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Error("%v", err)
			}
			break
		}
		log.Debug("%v", line)
	}
	end <- true
}
