 package main

 import (
	 "os"
         "fmt"
         "net"
         "bufio"
 )

var clientCount = 0
var allClients = make(map[net.Conn]int)

func sendDataToClients(msg string){

	// VRS ADSBx specific since no newline is printed between data bursts 
	// we use ] and must add } closure
	msg += "}"

	go func() {
		for incoming, _ := range allClients {
			_, err := incoming.Write([]byte(msg))

			if err != nil {
				fmt.Printf("Client %d (%s) disconnected \n", allClients[incoming], incoming.RemoteAddr().String())
				delete(allClients,incoming)
				clientCount -= 1
				fmt.Println("Current connected clients: ",clientCount)
				fmt.Println("Client Map: ", allClients)
				
			}

		}
	}()
} //end sendDataToClients

func handleConnection(incoming net.Conn) {

	go addtoConnMap(incoming) 
           
} //send handleConnection

func addtoConnMap(incoming net.Conn) {

        clientCount += 1
	allClients[incoming] = clientCount
	fmt.Println("Handling new connection... ",clientCount)
	fmt.Println("Client Map: ", allClients)

} //updateConnMap

func main() {

        hostName := "x.x.x.x"
	portNum := "xxxx"

	conn, err := net.Dial("tcp", hostName + ":" + portNum)
	// exit on TCP connect failure
	if err != nil {
       		fmt.Println(err)
		os.Exit(1)

        }
	
	// print error on listener error
	server, err := net.Listen("tcp", ":xxxx")
    	if err != nil {
       	 	fmt.Println("Listener ERR: %s",err)
		os.Exit(1)
    	}

	fmt.Printf("Connection established between %s and localhost.\n", hostName)
        fmt.Printf("Remote Address : %s \n", conn.RemoteAddr().String())
        fmt.Printf("Local Address : %s \n", conn.LocalAddr().String())


	// constantly accept JSON from PUB-VRS and write to the buffer
        go func(){
		for {
	            	status, err := bufio.NewReader(conn).ReadString(']')
	                     	if err != nil {
					//exit if can't read from TCP
                                	os.Exit(1)
	                	}
			
			//send data to connected clients using map
			go sendDataToClients(status)
		}
        }()


	for {

		incoming, err := server.Accept()
		// print error and continue waiting                
		if err != nil {
                        fmt.Println(err)
			incoming.Close()
			continue
                }
		fmt.Printf("Incoming Connection : %s \n", incoming.RemoteAddr().String())
		go handleConnection(incoming)

	}

}
