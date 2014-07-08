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
	targetProduction = flag.String("a", "localhost:8080", "Where production traffic goes. http://localhost:8080/production")
	altTarget        = flag.String("b", "localhost:8081", "Where testing traffic goes. Responses are ignored. http://localhost:8081/test")
)

type handler struct {
	Target      string
	Alternative string
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  if w == nil || req == nil || req.Body == nil {
    return
  }

	req1, req2 := DuplicateRequest(req)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in alternative; ", r)
			}
		}()

		client1 := &http.Client{}
		req1.URL.Host = h.Alternative
		req1.URL.Scheme = "http"
		resp, err := client1.Do(req1)
    req1.Body.Close()

    if resp != nil && resp.Body != nil {
      resp.Body.Close()
    }

		if err != nil {
			fmt.Printf("%s\n", err)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in target; ", r)
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
  req2.Body.Close()

  if resp != nil && resp.Body != nil {
    resp.Body.Close()
  }
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

	request1 = &http.Request{
		Method:        request.Method,
		URL:           &url.URL{
			Scheme: "http",
			Path: request.URL.Path,
			RawQuery: request.URL.RawQuery,
			Fragment: request.URL.Fragment,
		},
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        request.Header,
		Body:          nopCloser{b1},
		ContentLength: request.ContentLength,
	}
	request2 = &http.Request{
		Method:        request.Method,
		URL:           &url.URL{
			Scheme: "http",
			Path: request.URL.Path,
			RawQuery: request.URL.RawQuery,
			Fragment: request.URL.Fragment,
		},
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        request.Header,
		Body:          nopCloser{b2},
		ContentLength: request.ContentLength,
	}
	return
}
