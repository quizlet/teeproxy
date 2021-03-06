package main

import (
  "os"
  "fmt"
  "log"
  "testing"
  "net/http"
  "os/exec"
  "time"
  "io"
  "io/ioutil"
)

func answer200(port int) {
  log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func TestMemoryUse(t *testing.T) {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      fmt.Fprintf(w, "QZLT")
  })

  go answer200(8090)
  go answer200(8091)

  c := make(chan int)

  go func(c chan int) {
    cmd := exec.Command("./teeproxy", "-l", ":8092", "-a", "localhost:8091", "-b", "localhost:8090")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    err := cmd.Start()
    if err != nil {
      fmt.Fprintf(os.Stderr, "error starting teeproxy: %s\n", err)
      return
    }

    <-c

    cmd2 := fmt.Sprintf("/bin/ps -e -opid=,rss=,pmem=,args= | /usr/bin/grep './teeproxy'")
    str, err := exec.Command("/bin/sh", "-c", cmd2).Output()
    if err != nil {
      fmt.Fprintf(os.Stderr, "error getting mem: %s\n", err)
    }
    fmt.Printf("%d\n%s", cmd.Process.Pid, str)

    cmd.Process.Kill()
  }(c)

  time.Sleep(2 * time.Second)
  client := &http.Client{}

  for i := 0; i < 100000; i++ {
    resp, err := client.Get("http://0.0.0.0:8092/")

    if err != nil {
      fmt.Fprintf(os.Stderr, "error %d: %s\n", i, err)
    }

    if resp != nil && resp.Body != nil {
      io.Copy(ioutil.Discard, resp.Body)
      resp.Body.Close()
    }
  }

  c <- 1
  close(c)
  time.Sleep(1 * time.Second)
}

