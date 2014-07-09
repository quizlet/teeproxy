package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
  "io/ioutil"
	"net/http"
	"net/url"
	"runtime"
  "time"
  "log"
)

var (
	listen           = flag.String("l", ":8888", "port to accept requests")
	targetProduction = flag.String("a", "localhost:8080", "Where production traffic goes. http://localhost:8080/production")
	altTarget        = flag.String("b", "localhost:8081", "Where testing traffic goes. Responses are ignored. http://localhost:8081/test")
)

var httpclient *http.Client = &http.Client{}

func ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

		req1.URL.Host = *altTarget
		req1.URL.Scheme = "http"
		resp, err := httpclient.Do(req1)
    req1.Body.Close()

    if resp != nil && resp.Body != nil {
      io.Copy(ioutil.Discard, resp.Body)  // this copy is necessary for keepalives
      resp.Body.Close()
    }

		if err != nil {
			fmt.Printf("alt: %s\n", err)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in target; ", r)
		}
	}()

	req2.URL.Host = *targetProduction
	req2.URL.Scheme = "http"
	resp, err := httpclient.Do(req2)

	if err != nil {
		fmt.Printf("primary: %s\n", err)
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

  http.HandleFunc("/", ServeHTTP)

  s := &http.Server{
    Addr:           *listen,
    ReadTimeout:    5 * time.Second,
    WriteTimeout:   5 * time.Second,
  }
  log.Fatal(s.ListenAndServe())
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
