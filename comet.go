package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type Announcer struct {
	listeners map[chan string]bool
	sync.Mutex
}

func (a *Announcer) AnnounceLoop() {
	for {
		a.Lock()
		for listener, _ := range a.listeners {
			select {
			case listener <- "pong\n":
			case <-time.After(1 * time.Second):
				// shouldn't happen
				a.RemoveListener(listener)
			}
		}
		a.Unlock()
		time.Sleep(5 * time.Second)
	}
}

func (a *Announcer) AddListener(listener chan string) {
	a.Lock()
	defer a.Unlock()
	a.listeners[listener] = true
}

func (a *Announcer) RemoveListener(listener chan string) {
	a.Lock()
	defer a.Unlock()
	delete(a.listeners, listener)
}

func NewAnnouncer() *Announcer {
	a := &Announcer{}
	a.listeners = make(map[chan string]bool)
	return a
}

func FlushHTTP(rw *bufio.ReadWriter) {
	rw.WriteString("\r\n")
	rw.Flush()
}

func realtimeRoute(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Write(w)

	// listen to announcements
	ch := make(chan string)
	defer close(ch)
	mic.AddListener(ch)

	// hijack the connection
	if c, writer, err := w.(http.Hijacker).Hijack(); err != nil {
		return
	} else {
		defer c.Close()
		maxSessionLength := time.After(10 * time.Second)
		for {
			select {
			case msg := <-ch:
				_, l_err := fmt.Fprintf(writer, "%x\r\n", len(msg))
				if _, write_err := writer.WriteString(msg); l_err != nil || write_err != nil {
					mic.RemoveListener(ch)
					return
				}
				FlushHTTP(writer)
			case <-time.After(30 * time.Second):
				if _, write_err := writer.WriteString("\r\n"); write_err != nil {
					mic.RemoveListener(ch)
					return
				}
				FlushHTTP(writer)
			case <-maxSessionLength:
				fmt.Fprintf(writer, "%x\r\n", 0)
				FlushHTTP(writer)
				mic.RemoveListener(ch)
				return
			}
		}
		mic.RemoveListener(ch)
	}
	return
}

var mic *Announcer

func main() {
	m := http.NewServeMux()
	m.HandleFunc("/realtime", realtimeRoute)

	var port string = os.Getenv("PORT")
	if len(port) == 0 {
		port = "8000"
	}

	mic = NewAnnouncer()
	go mic.AnnounceLoop()

	fmt.Printf("comet chunk server at %v\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%v", port), m); err != nil {
		panic(err)
	}
}
