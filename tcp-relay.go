package main

 import (
         "fmt"
         "net"
         "bufio"
         "log"
         "os"
         "bytes"
 )

func main() {

        TCPoutput := make(chan string)
        var buffer bytes.Buffer

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
                                        
                       if err != nil {
                fmt.Println(err)
                return
        }

        server, err := net.Listen("tcp", ":6000")
        if err != nil {
            fmt.Println(err)
            os.Exit(1)
        }

        for {
                conn, err := server.Accept()
                if err != nil {
                           fmt.Println(err)
                           os.Exit(1)
                }
                for {
                         msg := <-TCPoutput
                         fmt.Fprintf(conn,msg)
                }
            }

}





