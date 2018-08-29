package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/cjkreklow/tcp-relay-pub-vrs/stream"
	"github.com/dustin/go-humanize"
)

var clients = struct {
	*sync.RWMutex
	*sync.WaitGroup
	total  int64
	active int64
	chans  map[int64]chan []byte
}{
	RWMutex:   new(sync.RWMutex),
	WaitGroup: new(sync.WaitGroup),
	chans:     make(map[int64]chan []byte),
}

var exitChannel = make(chan bool)

func runtimeStats(portNum string) {
	var m runtime.MemStats
	for {
		time.Sleep(5 * time.Second)
		fmt.Println(" =========  ==========  ==========")
		fmt.Println(portNum, " -  # goroutines: ", runtime.NumGoroutine(), " Client Connections: ", clients.active)
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

	var rMax = flag.Int("r", 0, "number of reconnect attempts")
	var rDelay = flag.Int("d", 10, "reconnect delay (in seconds)")

	flag.Parse()

	var args = flag.Args()

	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: tcp-relay-pub-vrs <hostname> <hostport> <relayport>")
		os.Exit(1)
	}

	var hostName = args[0]
	var portNum = args[1]
	var outportNum = args[2]

	var ch = make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go signalExitWatcher(ch)

	// start relay listener
	var server net.Listener
	server, err = net.Listen("tcp", net.JoinHostPort("", outportNum))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting relay: %s\n", err)
		os.Exit(1)
	}
	go func() {
		<-exitChannel
		server.Close()
		exit(-1)
	}()

	// start stream client
	var client = new(stream.StreamClient)
	client.ReconnectMax = *rMax
	client.ReconnectDelay = time.Duration(*rDelay) * time.Second
	err = client.Connect(hostName, portNum)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting upstream client: %s\n", err)
		os.Exit(1)
	}
	go func() {
		<-exitChannel
		client.Close()
		exit(-1)
	}()

	// start client receiving, discard messages if queue is full
	var msgChan = client.Scan(5, true)

	go dataSender(msgChan)
	go runtimeStats(portNum)

	for {
		var conn net.Conn
		conn, err = server.Accept()
		if err != nil {
			break
		}
		var ch = make(chan []byte, 2)
		clients.Lock()
		clients.total++
		clients.active++
		clients.chans[clients.total] = ch
		clients.Add(1)
		clients.Unlock()
		go clientWriter(clients.total, conn)
	}
	exit(2)
	clients.Wait()
	os.Exit(exitVal)
}

func dataSender(msgChan chan []byte) {
	var msg []byte
	var ok bool

	for {
		// non-blocking select, will pick up exit from after looping
		select {
		case <-exitChannel:
			clients.RLock()
			for _, ch := range clients.chans {
				close(ch)
			}
			clients.RUnlock()
			return
		default:
		}

		// blocking receive, expects msgChan will close on exit or error
		msg, ok = <-msgChan
		if !ok {
			exit(1)
			continue
		}
		clients.RLock()
		for _, ch := range clients.chans {
			// does not block on send, this will drop messages to
			// slow or dead receivers instead of backing up the
			// queue
			select {
			case ch <- msg:
			default:
			}
		}
		clients.RUnlock()
	}
}

func clientWriter(num int64, conn net.Conn) {
	var err error
	var msg []byte
	var ok bool

	var ch = clients.chans[num]

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
	clients.Lock()
	delete(clients.chans, num)
	clients.active--
	clients.Done()
	clients.Unlock()
}

// background goroutine to watch for OS exit signals and shutdown
func signalExitWatcher(c chan os.Signal) {
	// start graceful shutdown
	select {
	case <-c:
	case <-exitChannel:
	}

	exit(0)

	// second signal forces exit
	<-c
	os.Exit(1)
}

var exitOnce = new(sync.Once)
var exitVal int

// exit should be called on all exit paths to guarantee that the exit
// channel is closed and all goroutines have a chance to exit cleanly.
// The return value on the first call to exit will be used in the final
// call to os.Exit().
func exit(val int) {
	exitOnce.Do(func() {
		exitVal = val
		close(exitChannel)
		go func() {
			time.Sleep(30 * time.Second)
			fmt.Println("exit timeout")
			os.Exit(-2)
		}()
	})
}
