package main

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/cjkreklow/tcp-relay-pub-vrs/stream"
	"github.com/dustin/go-humanize"
)

var clientCount int64
var allClients = make(map[int64]chan []byte)

func runtimeStats(portNum string) {
	var m runtime.MemStats
	for {
		time.Sleep(5 * time.Second)
		fmt.Println(" =========  ==========  ==========")
		fmt.Println(portNum, " -  # goroutines: ", runtime.NumGoroutine(), " Client Connections: ", clientCount)
		runtime.ReadMemStats(&m)
		fmt.Println("Memory Acquired: ", humanize.Bytes(m.Sys))
		//fmt.Println("Memory Alloc: ", humanize.Bytes(m.Alloc))
		fmt.Println("GC: ", m.EnableGC)
		//fmt.Println("# GC: ", m.NumGC)
		fmt.Println("Last GC: ", m.LastGC)
		fmt.Println("Next GC: ", humanize.Bytes(m.NextGC), "Heap Alloc: ", humanize.Bytes(m.HeapAlloc))
		//fmt.Printf("%d, %d, %d, %d\n", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.HeapReleased)
		fmt.Println(" =========  ==========  ==========")
		//runtime.GC()
	}
}

func main() {
	var err error

	if len(os.Args) != 4 {
		fmt.Println("usage: tcp-relay-pub-vrs <hostname> <hostport> <relayport>")
		os.Exit(1)
	}

	var hostName = os.Args[1]
	var portNum = os.Args[2]
	var outportNum = os.Args[3]

	// start relay listener
	var server net.Listener
	server, err = net.Listen("tcp", net.JoinHostPort("", outportNum))
	if err != nil {
		fmt.Printf("error starting relay: %s\n", err)
		os.Exit(1)
	}
	defer server.Close()

	// start stream client
	var client = new(stream.StreamClient)
	err = client.Connect(hostName, portNum)
	if err != nil {
		fmt.Printf("error starting client: %s\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// start client receiving, discard messages if queue is full
	var msgChan = client.Scan(5, true)

	go dataSender(msgChan)
	go runtimeStats(portNum)

	var clientNum int64
	var clientWg = new(sync.WaitGroup)
	for {
		var clientConn net.Conn
		clientConn, err = server.Accept()
		if err != nil {
			continue
		}
		var clientChan = make(chan []byte, 2)
		clientCount++
		clientNum++
		allClients[clientNum] = clientChan
		clientWg.Add(1)
		go clientWriter(clientNum, clientConn, clientChan, clientWg)
	}
}

func dataSender(msgChan chan []byte) {
	var msg []byte
	var ok bool

	for {
		select {
		case msg, ok = <-msgChan:
			if !ok {
				for _, ch := range allClients {
					close(ch)
				}
				return
			}
		}
		for _, ch := range allClients {
			select {
			case ch <- msg:
			default:
			}
		}
	}
}

func clientWriter(num int64, conn net.Conn, ch chan []byte, wg *sync.WaitGroup) {
	var err error
	var msg []byte
	var ok bool

Loop:
	for {
		select {
		case msg, ok = <-ch:
			if !ok {
				break Loop
			}
			_, err = conn.Write(msg)
			if err != nil {
				break Loop
			}
		}
	}
	conn.Close()
	delete(allClients, num)
	clientCount--
	wg.Done()
}
