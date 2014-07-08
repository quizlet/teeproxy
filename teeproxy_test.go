package main

import (
  "os"
  "fmt"
  "log"
  "testing"
  "net/http"
  "os/exec"
  "time"
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
    //stdout := cmd.StdoutPipe()
    err := cmd.Start()
    if err != nil {
      fmt.Fprintf(os.Stderr, "error starting teeproxy: %s\n", err)
    }

    <-c

    //cmd2 := fmt.Sprintf("/bin/ps -e -opid=,rss=,pmem= | /usr/bin/grep %d", cmd.Process.Pid)
    cmd2 := fmt.Sprintf("/bin/ps -e -opid=,rss=,pmem=,args= | /usr/bin/grep './teeproxy'")
    str, err := exec.Command("/bin/sh", "-c", cmd2).Output()
    if err != nil {
      fmt.Fprintf(os.Stderr, "error getting mem: %s\n", err)
    }
    fmt.Printf("%d\n%s", cmd.Process.Pid, str)

    cmd.Process.Kill()
  }(c)

  time.Sleep(2 * time.Second)

  for i := 0; i < 10000; i++ {
    _, err := http.Get("http://0.0.0.0:8092/")

    if err != nil {
      fmt.Fprintf(os.Stderr, "error %d: %s\n", i, err)
    }
  }

  c <- 1
  time.Sleep(1 * time.Second)
}

