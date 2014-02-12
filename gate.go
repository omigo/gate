package main

import (
	"./spdy"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

var end chan bool

func main() {
	rawurl := flag.String("u", "", "Rawurl")
	data := flag.String("d", "", "verbose 3")
	times := flag.Int("t", 1, "verbose 3")
	verbose1 := flag.Bool("v", false, "verbose 1")
	verbose2 := flag.Bool("vv", false, "verbose 2")

	flag.Parse()

	if *rawurl == "" {
		fmt.Println("Rawurl must not blank")
		os.Exit(1)
	}

	level := byte(3)

	if *verbose2 {
		level = 1
	} else if *verbose1 {
		level = 2
	}

	log := spdy.GetLogger()
	log.SetLevel(level)

	var req *http.Request
	var err error

	if *data == "" {
		req, err = http.NewRequest("GET", *rawurl, nil)
	} else {
		req, err = http.NewRequest("POST", *rawurl, bytes.NewBufferString(*data))
	}
	if err != nil {
		log.Error("%v", err)
	}

	req.Header.Set("Cache-Control", "nostore")
	req.Header.Set("accept-encoding", "gzip, deflate")
	req.Header.Set("user-agent", "gate/0.0.1")

	end = make(chan bool, *times)
	for i := 0; i < *times; i++ {
		id, err := spdy.Request(req, handle)
		if err != nil {
			log.Error("%v", err)
		}
		log.Debug("Id#%d is sent", id)
	}

	for i := 0; i < *times; i++ {
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

	if res.Body != nil {

		rd := bufio.NewReader(res.Body)
		defer res.Body.Close()
		for {
			line, err := rd.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Printf("%v", line)
				} else {
					fmt.Printf("%v", err)
				}
				break
			}
			fmt.Printf("%v", line)
		}
	}

	end <- true
}
