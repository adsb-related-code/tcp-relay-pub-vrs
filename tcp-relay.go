package main

 import (
         "fmt"
         "net"
         "bufio"
         "log"
         "os"
  //       "bytes"
 )

var TCPoutput = make(chan string)
var buffer bytes.Buffer
var clientCount = 0
var allClients = make(map[net.Conn]int)

func sendDataToClients(msg string){

        //specific to ASDBx
        //json object ]
        //ReadString 
        msg += "}"
 
        go func(){
                for conn, _ := range allClients {

                              //_, err := fmt.Fprintf(conn,"%s",msg)
                                _, err := conn.Write([]byte(msg))


                                if err != nil {
                                                delete(allClients,conn)
                                                clientCount -= 1
                                }
                                
                }
         }()

      //clean up memory usage
      //really should pull this out so I'm not writing to buffer and clearing it at the same time
      //anyone help with this?
      //buffer.Reset()
}

func handleConnection(conn net.Conn) {

        clientCount += 1
        allClients[conn] = clientCount
        //fmt.Println("Handling new connection... ",clientCount)

}


func main() {

        //public ASBX feed
        hostName := "pub-vrs.adsbexchange.com"
        portNum := "32005"

        conn, err := net.Dial("tcp", hostName + ":" + portNum)
        if err != nil {
                fmt.Println(err)
                os.Exit(1)
        }
 
       /*
       // going to modify for multiple listeners?!
       // tcp server can push to this server instead of this server pulling
       // for future design
       
       conn, err := net.Listen("tcp", ":32001")
        if err != nil {
            fmt.Println(err)
            os.Exit(1)
        }
       */
       go func() {

                fmt.Printf("Connection established between %s and localhost.\n", hostName)
                fmt.Printf("Remote Address : %s \n", conn.RemoteAddr().String())
                fmt.Printf("Local Address : %s \n", conn.LocalAddr().String())
        
                // Add more statistics periodically printed out
                //  go func() printing to terminal or outputin json on a port

                for {
                        status, err := bufio.NewReader(conn).ReadString(']')

                        if err != nil {
                                log.Fatal(err)
                        }

                      go sendDataToClients(status)

                }

        }()

        server, err := net.Listen("tcp", ":6000")
        if err != nil {
            fmt.Println(err)
            os.Exit(1)
        }

        for {
                
                conn, err := server.Accept()
                if err != nil {
                        fmt.Println(err)
                        }

                go handleConnection(conn)
  
          }

}





