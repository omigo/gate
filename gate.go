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
	rawurl := "https://127.0.0.1:2443"
//	rawurl := "https://isspdyenabled.com/"
	verbose := "vvv"

	level := byte(3 - len(verbose))
	log := spdy.GetLogger()
	log.SetLevel(level)

	req, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		log.Error("%v", err)
	}

	req.Header.Set("Cache-Control","nostore")
	req.Header.Set("accept-encoding", "gzip, deflate")
	req.Header.Set("user-agent", "gate/0.0.1")

	for i := 0; i < 1; i++ {
		res := spdy.Request(req)

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
	}
}
