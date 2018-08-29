package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"runtime"
	"sync"

	"github.com/dustin/go-humanize"
)

var clientCount = 0
var allClients = make(map[net.Conn]int)
var connLock sync.RWMutex

func runtimeStats(portNum string) {
	var m runtime.MemStats
	time.Sleep(3 * time.Second)
	for {
		runtime.ReadMemStats(&m)
		fmt.Println()
		fmt.Printf("Goroutines:\t%7d\t\t Clients:\t%7d\n", runtime.NumGoroutine(), clientCount)
		fmt.Printf("Last GC:\t%7d\t\t Next GC:\t%7s\n", m.LastGC, humanize.Bytes(m.NextGC))
		fmt.Printf("Memory Acq:\t%7s\t\t Heap Alloc:\t%7s\n", humanize.Bytes(m.HeapSys), humanize.Bytes(m.HeapAlloc))
		fmt.Printf("GC Pool Mem:\t%7s\t\t Heap InUse:\t%7s\n", humanize.Bytes(m.HeapIdle-m.HeapReleased), humanize.Bytes(m.HeapInuse))
		fmt.Println()
		time.Sleep(5 * time.Second)
	}
}

func sendDataToClient(client net.Conn, msg string) {
	n, err := client.Write([]byte(msg))
	if err != nil {
		log.Printf("Client %s disconnected \n", client.RemoteAddr().String())
		removeFromConnMap(client)
	} else if n != len(msg) {
		log.Printf("Client connection did not send expected number of bytes, %d != %d", n, len(msg))
	}
}

func sendDataToClients(msg string) {
	// VRS ADSBx specific since no newline is printed between data bursts
	// we use ] and must add } closure
	msg += "}"

	connLock.RLock()
	for client := range allClients {
		go sendDataToClient(client, msg)
	}
	connLock.RUnlock()
}

func removeFromConnMap(client net.Conn) {
	connLock.Lock()
	log.Println("removing client", clientCount)
	delete(allClients, client)
	clientCount = len(allClients)
	connLock.Unlock()
}

func addToConnMap(client net.Conn) {
	connLock.Lock()
	log.Println("adding client", clientCount+1)
	allClients[client] = 0
	clientCount = len(allClients)
	connLock.Unlock()
}

func handleTCPIncoming(hostName string, portNum string) {
	conn, err := net.Dial("tcp", hostName+":"+portNum)
	// exit on TCP connect failure
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// constantly read JSON from PUB-VRS and write to the buffer
	data := bufio.NewReader(conn)
	for {
		scan, err := data.ReadString(']')
		if len(scan) == 0 || err != nil {
			break
		}

		go sendDataToClients(scan)
	}
}

func handleTCPOutgoing(outportNum string) {
	// print error on listener error
	server, err := net.Listen("tcp", ":"+outportNum)
	if err != nil {
		log.Fatalf("Listener err: %s\n", err)
	}

	for {
		incoming, err := server.Accept()
		// print error and continue waiting
		if err != nil {
			log.Println(err)
		} else {
			go addToConnMap(incoming)
		}

	}
}

func main() {
	hostName := flag.String("hostname", "", "what host to connect to")
	portNum := flag.String("port", "", "which port to connect with")
	outportNum := flag.String("listenport", "", "which port to listen on")
	flag.Parse()

	if *hostName == "" || *portNum == "" || *outportNum == "" {
		fmt.Println("usage: dial-tcp -hostname <input> -port <port> -listenport <output>")
		os.Exit(1)
	}

	go runtimeStats(*portNum)
	go handleTCPOutgoing(*outportNum)

	// if this function returns, the main thread will exit, which exits the entire program
	handleTCPIncoming(*hostName, *portNum)
}
