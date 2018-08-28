 package main

 import (
	 "os"
         "fmt"
         "net"
//	"io"
         "bufio"
	 "time"
//	"bytes"
//	"runtime/debug"
	"runtime"
	"github.com/dustin/go-humanize"
 )

var clientCount = 0
var allClients = make(map[net.Conn]int)

func runtimeStats(portNum string){
	var m runtime.MemStats
	for {
		time.Sleep(5 * time.Second)
		fmt.Println("# goroutines: ", runtime.NumGoroutine()," Client Connections: ", clientCount)
		runtime.ReadMemStats(&m)
		fmt.Println("Memory Acquired: ", humanize.Bytes(m.Sys))
		//fmt.Println("Memory Alloc: ", humanize.Bytes(m.Alloc))
		//fmt.Println("GC: ", m.EnableGC)
		//fmt.Println("# GC: ", m.NumGC)
		fmt.Println("Last GC: ", humanize.Bytes(m.LastGC))
		fmt.Println("Next GC: ", humanize.Bytes(m.NextGC))
		//fmt.Printf("%d, %d, %d, %d\n", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.HeapReleased)
	}

}

func sendDataToClients(msg string){

	// VRS ADSBx specific since no newline is printed between data bursts 
	// we use ] and must add } closure
	//go func() {
		msg += "}"
		
		for incoming, _ := range allClients {
			
			go func() {			
				_, err := incoming.Write([]byte(msg))
				if err != nil {
					fmt.Printf("Client %d (%s) disconnected \n", allClients[incoming], incoming.RemoteAddr().String())
					delete(allClients,incoming)
					clientCount -= 1		
				} 

			//_, err := io.WriteString(incoming, msg)
			//_, err := incoming.Write([]byte(msg))
			//if err != nil {
				//fmt.Println(time.Now().Format(time.RFC850))
				//fmt.Printf("Client %d (%s) disconnected \n", allClients[incoming], incoming.RemoteAddr().String())
			//	delete(allClients,incoming)
			//	clientCount -= 1
				//fmt.Println("Current connected clients: ",clientCount)
				//fmt.Println("Client Map: ", allClients)
			//	incoming.Close()
			//}
			}()
		}
		msg = ""
		
	//debug.FreeOSMemory()
	//}()
} //end sendDataToClients


func addtoConnMap(incoming net.Conn) {
	//fmt.Println(time.Now().Format(time.RFC850))
        clientCount += 1
	allClients[incoming] = clientCount
	fmt.Println("Handling new connection... ",clientCount," from ", incoming.RemoteAddr().String())
	//fmt.Println("Client Map: ", allClients)

} //addtoConnMap

func handleTCPincoming(hostName string, portNum string){

	conn, err := net.Dial("tcp", hostName + ":" + portNum)
	// exit on TCP connect failure
	if err != nil {
       		//fmt.Println(err)
		conn.Close()
		os.Exit(1)

        }
	defer conn.Close()

        fmt.Printf("Remote Address : %s \n", conn.RemoteAddr().String())
        fmt.Printf("Local Address : %s \n", conn.LocalAddr().String())


	//go runtimeStats()
	// constantly read JSON from PUB-VRS and write to the buffer
        //go func(){

	data := bufio.NewReader(conn)
	for {
            	scan, err := data.ReadString(']')
                     	if err != nil {
				//exit if can't read from TCP
				conn.Close()
                        	os.Exit(1)
                	} else {
				//send data to connected clients using map
				go sendDataToClients(scan)
				scan = ""
			}
	}
	
	//}()
}


func main() {
        	hostName := ""
		portNum := ""
		outportNum := ""

	if(len(os.Args) > 2 ){
        	hostName = os.Args[1]
		portNum = os.Args[2]
		outportNum = os.Args[3]
	} else {
		fmt.Println("usage: dial-tcp <input> <port> <output>")
		os.Exit(1)
	}

	//connect to TCP data for sending to clients	
	go handleTCPincoming(hostName,portNum)

	//go runtimeStats(portNum)

	// print error on listener error
	server, err := net.Listen("tcp", ":"+outportNum)
    	if err != nil {
       	 	fmt.Println("Listener ERR: %s",err)
		os.Exit(1)
    	}

	for {
		incoming, err := server.Accept()
		// print error and continue waiting
		if err != nil {
                        fmt.Println(err)
			incoming.Close()
                } else {

		        go addtoConnMap(incoming)

		}
		defer incoming.Close()
	}

}
