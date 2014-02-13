package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"github.com/gavinsh/gate/spdy"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

var end chan bool
var quiet bool

func main() {
	rawurl := flag.String("u", "", "Raw url")
	data := flag.String("d", "", "POST data")
	times := flag.Int("t", 1, "Request times")
	verbose1 := flag.Bool("v", false, "Verbose")
	verbose2 := flag.Bool("vv", false, "Verbose detail")
	quieta := flag.Bool("q", false, "Quiet")

	flag.Parse()

	quiet = *quieta

	if *rawurl == "" {
		fmt.Println("Raw url must not blank")
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

	req.Header.Set("user-agent", "gate/0.1.0")

	if quiet {
		dump, _ := httputil.DumpRequest(req, false)
		fmt.Println(string(dump))
	}

	end = make(chan bool, *times)

	fmt.Printf("Init  %v\n", time.Now())
	id, err := spdy.Request(req, handle)
	if err != nil {
		log.Error("%v", err)
	}
	defer spdy.Close()
	log.Debug("Id#%d is sent", id)

	t1 := time.Now()
	fmt.Printf("Start %v\n", t1)
	for i := *times - 1; i > 0; i-- {
		id, err := spdy.Request(req, handle)
		if err != nil {
			log.Error("%v", err)
		}
		log.Debug("Id#%d is sent", id)
	}
	for i := *times; i > 0; i-- {
		<-end
	}

	t2 := time.Now()
	fmt.Printf("\n\nEnd   %v\n", t2)
	fmt.Printf("\nRequest %d times(exclude init Session) use %.3fs.\n", *times, (float64(t2.Sub(t1)))/1e9)
}

func handle(streamId uint32, res *http.Response, err error) {
	go func(){
		defer func() {
			end <- true
		}()

		if err != nil {
			fmt.Printf("< %v", err)
		}

		if quiet {
			io.Copy(ioutil.Discard, res.Body)
			return
		}

		fmt.Printf("\nStreamId#%d: \n", streamId)

		dump, _ := httputil.DumpResponse(res, false)
		fmt.Println(string(dump))

		if res.Body != nil {
			if len(res.Header["Content-Encoding"]) > 0 &&
			res.Header["Content-Encoding"][0] == "gzip" {
				res.Body, _ = gzip.NewReader(res.Body) // err
			}
			r := bufio.NewReader(res.Body)
			defer res.Body.Close()
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						fmt.Printf("%5d: %v", streamId, line)
					} else {
						fmt.Printf("%v", err)
					}
					break
				}
				fmt.Printf("%5d: %v", streamId, line)
			}
		}
	}()
}
