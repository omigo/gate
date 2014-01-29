package main

import (
	"bufio"
	"github.com/gavines/gate/spdy"
	"io"
	"net/http"
)

func main() {
	rawurl := "https://10.15.107.172/index.html"
	verbose := "vvv"

	level := byte(4 - len(verbose))
	log := spdy.GetLogger()
	log.SetLevel(level)

	req, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		log.Error("%v", err)
	}

	req.Header.Set("accept-encoding", "gzip, deflate")
	req.Header.Set("user-agent", "gate/0.0.1")

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
