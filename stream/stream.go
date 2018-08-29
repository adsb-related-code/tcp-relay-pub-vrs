// Copyright 2018 Collin Kreklow
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
// BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
// ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package stream

import (
	"errors"
	"fmt"
	"net"
	"time"
)

// StreamClient connects to a data stream and splits the stream into
// JSON messages.
type StreamClient struct {
	// Reconnection options
	ReconnectMax   int
	ReconnectDelay time.Duration

	// Err contains any error returned by the scanner
	Err error

	addr       string
	conn       net.Conn
	reconCount int

	scanner *scanner

	msgChan chan []byte

	closed bool
}

// Connect establishes a connection to the server.
func (sc *StreamClient) Connect(server, port string) error {
	var err error

	if sc.closed {
		return errors.New("cannot connect closed client")
	}

	sc.addr = net.JoinHostPort(server, port)
	sc.conn, err = net.DialTimeout(
		"tcp",
		sc.addr,
		(30 * time.Second),
	)
	if err != nil {
		return err
	}

	return nil
}

// Close closes the network connection. An attempt is made to signal the
// scanner to stop scanning at the end of the current message.
func (sc *StreamClient) Close() {
	if sc.closed {
		return
	}

	sc.closed = true
	if sc.scanner != nil {
		sc.scanner.stop = true
	}
	if sc.conn != nil {
		time.Sleep(500 * time.Millisecond)
		sc.conn.Close()
	}
}

// Scan initates decoding of the stream and returns a message channel.
// The argument passsed is the size of the message channel. A nil
// message indicates an error, receivers should check StreamClient.Err
// for the last error returned by the underlying stream scanner.
func (sc *StreamClient) Scan(size int, discard bool) chan []byte {
	if sc.closed {
		sc.Err = errors.New("cannot scan closed client")
		return nil
	}

	sc.scanner = newStreamScanner(sc.conn, 5*1024*1024)
	sc.msgChan = make(chan []byte, size)
	go sc.scanAndWatch(discard)
	return sc.msgChan
}

// scanAndWatch sends incoming messages to the message channel and
// reconnects if an error occurs.
func (sc *StreamClient) scanAndWatch(discard bool) {
	var err error

	if sc.closed {
		return
	}

	defer func() {
		sc.closed = true
		sc.Err = sc.scanner.Err()
		close(sc.msgChan)
	}()

	for {
		for sc.scanner.Scan() { // Scan() returns false on error
			if sc.scanner.Bytes() == nil {
				continue
			}
			select {
			case sc.msgChan <- append([]byte{}, sc.scanner.Bytes()...):
				continue
			default:
				if discard {
					continue
				}
				sc.msgChan <- append([]byte{}, sc.scanner.Bytes()...)
			}
		}
		if sc.closed { // Close() was called on the client
			return
		}
		sc.conn.Close() // close the existing connection
		if sc.ReconnectMax > 0 {
			if sc.ReconnectDelay == 0 {
				sc.ReconnectDelay = time.Second
			}
			for {
				time.Sleep(sc.ReconnectDelay)
				fmt.Println("attempting to reconnect...")
				sc.conn, err = net.DialTimeout(
					"tcp",
					sc.addr,
					(30 * time.Second),
				)
				if err != nil {
					if sc.reconCount >= sc.ReconnectMax {
						fmt.Println("failed to reconnect")
						return
					}
					sc.reconCount++
					continue
				}
				fmt.Println("reconnected")
				sc.scanner = newStreamScanner(sc.conn, 5*1024*1024)
				sc.reconCount = 0
				break
			}
			continue
		}
		break
	}
}
