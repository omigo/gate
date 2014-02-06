package main

import (
	"./spdy"
	"bufio"
	"fmt"
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

	times := 1
	for i := 0; i < times; i++ {
		id, err := spdy.Request(req, handle)
		if err != nil {
			log.Error("%v", err)
		}
		log.Info("Id#%d is sent", id)
	}
}

func handle(streamId uint32, res *http.Response, err error) {
	if err != nil {
		fmt.Println("< %v", err)
	}

	fmt.Println("StreamId#%d: ", streamId)

	for k, vs := range res.Header {
		fmt.Println("%-32s%s", k, vs)
	}

	rd := bufio.NewReader(res.Body)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println("%v", err)
			}
			break
		}
		fmt.Println("%v", line)
	}
}
