package main

import (
	"fmt"
	"net"
	"os"
	"io"
	"bufio"
	"time"
	//	"bytes"
	//"runtime/debug"
	//	"strings"
	"github.com/dustin/go-humanize"
	"runtime"
	"sync"
	
)

var clientCount = 0
var allClients = make(map[net.Conn]int)
var connLock sync.RWMutex

func runtimeStats(portNum string) {
	var m runtime.MemStats
	for {
		time.Sleep(5 * time.Second)
		fmt.Println(" =========  ==========  ==========")
		fmt.Println(portNum," -  # goroutines: ", runtime.NumGoroutine(), " Client Connections: ", clientCount)
		runtime.ReadMemStats(&m)
		fmt.Println("Memory Acquired: ", humanize.Bytes(m.Sys))
		//fmt.Println("Memory Alloc: ", humanize.Bytes(m.Alloc))
		fmt.Println("GC: ", m.EnableGC)
		//fmt.Println("# GC: ", m.NumGC)
		fmt.Println("Last GC: ", m.LastGC)
		fmt.Println("Next GC: ", humanize.Bytes(m.NextGC), "Heap Alloc: ",  humanize.Bytes(m.HeapAlloc))
		//fmt.Printf("%d, %d, %d, %d\n", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.HeapReleased)
		fmt.Println(" =========  ==========  ==========")
		//runtime.GC()
	}
	return

}

func sendDataToClients(msg string){

	// VRS ADSBx specific since no newline is printed between data bursts
	// we use ] and must add } closure
	msg += "}"

	//var wg sync.WaitGroup

	for incoming, _ := range allClients {

		//wg.Add(1)

		go func() {		
			_, err := incoming.Write([]byte(msg))
			if err != nil {
			fmt.Printf("Client %d (%s) disconnected \n", allClients[incoming], incoming.RemoteAddr().String())
				go removefromConnMap(incoming)
			}
			fmt.Printf(".")
			//defer wg.Done()
		}()
		
	}
	//wg.Wait()
	//fmt.Println(" ========= CALLING FREE OS MEMORY ========== ")
	//debug.FreeOSMemory()
	//}()
} //end sendDataToClients

func removefromConnMap(incoming net.Conn) {

	//onnLock.Lock()
	//defer connLock.Unlock()	
	delete(allClients, incoming)
	clientCount = len(allClients)
	//fmt.Println("disconnected connection ...")

} //removefromConnMap

func addtoConnMap(incoming net.Conn) {
	//connLock.Lock()
	//defer connLock.Unlock()	
	allClients[incoming] = clientCount+1
	clientCount = len(allClients)
	//fmt.Println("incoming connection ...")
} //addtoConnMap

func handleTCPincoming(hostName string, portNum string) {

	conn, err := net.Dial("tcp", hostName+":"+portNum)
	// exit on TCP connect failure
	if err != nil {
		conn.Close()
		os.Exit(1)
	}
	//	defer conn.Close()

	//fmt.Printf("Remote Address : %s \n", conn.RemoteAddr().String())
	//fmt.Printf("Local Address : %s \n", conn.LocalAddr().String())

	//go runtimeStats()
	// constantly read JSON from PUB-VRS and write to the buffer
	data := bufio.NewReader(conn)
	for {
		scan, err := data.ReadString(']')
		if len(scan) == 0 && err != nil {
            		if err == io.EOF {
                		break
            			}
            		os.Exit(1)
       		}
		
		go sendDataToClients(scan)
		
	}

	fmt.Println("THIS SHOULD NEVER PRINT!")
	
}

func main() {
	hostName := ""
	portNum := ""
	outportNum := ""

	

	if len(os.Args) > 2 {
		hostName = os.Args[1]
		portNum = os.Args[2]
		outportNum = os.Args[3]
	} else {
		//fmt.Println("usage: dial-tcp <input> <port> <output>")
		os.Exit(1)
	}

	//connect to TCP data for sending to clients
	go handleTCPincoming(hostName, portNum)
	//go runtimeStats(portNum)

	// print error on listener error
	server, err := net.Listen("tcp", ":"+outportNum)
	if err != nil {
		//fmt.Println("Listener ERR: %s", err)
		os.Exit(1)
	}

	for {
		incoming, err := server.Accept()
		// print error and continue waiting
		if err != nil {
			//fmt.Println(err)
			break
		} else {
			//fmt.Println("adding Client ...")
			//go addtoConnMap(incoming)
			go addtoConnMap(incoming)
		}

	}

	

}
