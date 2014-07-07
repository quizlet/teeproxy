package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
)

var (
	listen           = flag.String("l", ":8888", "port to accept requests")
	targetProduction = flag.String("a", "localhost:8080", "where production traffic goes. http://localhost:8080/production")
	altTarget        = flag.String("b", "localhost:8081", "where testing traffic goes. response are skipped. http://localhost:8081/test")
	debug            = flag.Bool("debug", false, "more logging, showing ignored output")
)

type handler struct {
	Target      string
	Alternative string
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req1, req2 := DuplicateRequest(req)
	go func() {
		defer func() {
			if r := recover(); r != nil && *debug {
				fmt.Println("Recovered in f", r)
			}
		}()

		client1 := &http.Client{}
		req1.URL.Host = h.Alternative
		req1.URL.Scheme = "http"
		_, err := client1.Do(req1)

		if err != nil {
			fmt.Printf("%s\n", err)
		}
	}()
	defer func() {
		if r := recover(); r != nil && *debug {
			fmt.Println("Recovered in f", r)
		}
	}()

	client2 := &http.Client{}
	req2.URL.Host = h.Target
	req2.URL.Scheme = "http"
	resp, err := client2.Do(req2)

	if err != nil {
		fmt.Printf("%s\n", err)
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	local, _ := net.Listen("tcp", *listen)
	h := handler{
		Target:      *targetProduction,
		Alternative: *altTarget,
	}
	http.Serve(local, h)
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func DuplicateRequest(request *http.Request) (request1 *http.Request, request2 *http.Request) {
	b1 := new(bytes.Buffer)
	b2 := new(bytes.Buffer)
	w := io.MultiWriter(b1, b2)
	io.Copy(w, request.Body)
	defer request.Body.Close()
	request1 = &http.Request{
		Method:        request.Method,
		URL:           &url.URL{Scheme: "http"},
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        request.Header,
		Body:          nopCloser{b1},
		ContentLength: request.ContentLength,
	}
	request2 = &http.Request{
		Method:        request.Method,
		URL:           &url.URL{Scheme: "http"},
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        request.Header,
		Body:          nopCloser{b2},
		ContentLength: request.ContentLength,
	}
	return
}
