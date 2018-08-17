package main

import (
    "bufio"
    "fmt"
    "net"
    "os"
)

  func main() {

    clientCount := 0
    allClients := make(map[net.Conn]int)
    newConnections := make(chan net.Conn)
    deadConnections := make(chan net.Conn)
    messages := make(chan string)

    server, err := net.Listen("tcp", ":6000")
    if err != nil {
            fmt.Println(err)
            os.Exit(1)
    }

    go func() {
            for {
                    conn, err := server.Accept()
                    if err != nil {
                            fmt.Println(err)
                            os.Exit(1)
                    }
                    newConnections <- conn
            }
    }()

    for {

            select {
            case conn := <-newConnections:

                    allClients[conn] = clientCount
                    clientCount += 1

                    go func(conn net.Conn, clientId int) {
                            scanner := bufio.NewScanner(conn)
                            for scanner.Scan() {
                                    if err := scanner.Err(); err != nil {
                                      break
                                    }
                                    messages <- scanner.Text()

                            }
                            deadConnections <- conn


                    }(conn, allClients[conn])

            case message := <-messages:

                    for conn, _ := range allClients {

                            go func(conn net.Conn, message string) {
                                    _, err := conn.Write([]byte(message))
                                    if err != nil {
                                            deadConnections <- conn
                                    }
                            }(conn, message)
                    }
            case conn := <-deadConnections:
                    delete(allClients, conn)
            }
        }
  }
