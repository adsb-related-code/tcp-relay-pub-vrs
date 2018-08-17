package main

 import (
         "fmt"
         "net"
         "bufio"
         "log"
 //        "os"
         "bytes"
 )

var TCPoutput = make(chan string)
var buffer bytes.Buffer

func handleConnection(conn net.Conn) {

        fmt.Println("Handling new connection...")

        // Close connection when this function ends

        defer func() {
                fmt.Println("Closing connection...")
                conn.Close()
        }()

        for {
            msg := <-TCPoutput
            fmt.Fprintf(conn,msg)
        }
}


func main() {

        //public ASBX feed
        hostName := "pub-vrs.adsbexchange.com"
        portNum := "32005"

        conn, err := net.Dial("tcp", hostName + ":" + portNum)
        if err != nil {
                fmt.Println(err)
                return
        }


        go func() {

                fmt.Printf("Connection established between %s and localhost.\n", hostName)
                fmt.Printf("Remote Address : %s \n", conn.RemoteAddr().String())
                fmt.Printf("Local Address : %s \n", conn.LocalAddr().String())

                for {
                        status, err := bufio.NewReader(conn).ReadString(']')

                        if err != nil {
                                log.Fatal(err)
                        }

                        buffer.WriteString(status)
                        buffer.WriteString("}")
                        TCPoutput <- buffer.String()
                        buffer.Reset()
                }

        }()

        server, err := net.Listen("tcp", ":6000")
        if err != nil {
            fmt.Println(err)
        }

        for {
                
                conn, err := server.Accept()
                if err != nil {
                        fmt.Println(err)
                        break
                }

                go handleConnection(conn)
  
          }

}





