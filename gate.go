package main

import (
	"./spdy"
	"bufio"
	"fmt"
	"io"
	"net/http"
)

var end chan bool

func main() {
	//	rawurl := "https://www.google.com"
	//	rawurl := "https://wordpress.com"
	rawurl := "http://10.15.107.172:2800/index.html"
	//rawurl := "http://127.0.0.1:2800/index.html"
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
	end = make(chan bool, times)
	for i := 0; i < times; i++ {
		id, err := spdy.Request(req, handle)
		if err != nil {
			log.Error("%v", err)
		}
		log.Info("Id#%d is sent", id)
	}

	for i := 0; i < times; i++ {
		<-end
	}

	spdy.Close()
}

func handle(streamId uint32, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("< %v", err)
	}

	fmt.Printf("StreamId#%d: \n", streamId)

	for k, vs := range res.Header {
		fmt.Printf("%-32s%s\n", k+":", vs)
	}
	fmt.Println()

	rd := bufio.NewReader(res.Body)
	defer res.Body.Close()
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Printf("%v", err)
			}
			break
		}
		fmt.Printf("%v", line)
	}

	end <- true
}
