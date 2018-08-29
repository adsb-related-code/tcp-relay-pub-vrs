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
	"net"
	"time"
)

// StreamClient connects to a data stream and splits the stream into
// JSON messages.
type StreamClient struct {
	// Err contains any error returned by the scanner
	Err error

	conn    net.Conn
	scanner *scanner
}

// Connect establishes a connection to the server.
func (sc *StreamClient) Connect(server, port string) error {
	var err error

	sc.conn, err = net.DialTimeout(
		"tcp",
		net.JoinHostPort(server, port),
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
	sc.scanner.stop = true
	time.Sleep(500 * time.Millisecond)
	sc.conn.Close()
}

// Scan initates decoding of the stream and returns a message channel.
// The argument passsed is the size of the message channel. A nil
// message indicates an error, receivers should check StreamClient.Err
// for the last error returned by the underlying stream scanner.
func (sc *StreamClient) Scan(size int, discard bool) chan []byte {
	sc.scanner = newStreamScanner(sc.conn, 5*1024*1024)
	var ch = make(chan []byte, size)
	go func() {
		for sc.scanner.Scan() {
			if sc.scanner.Bytes() == nil {
				continue
			}
			select {
			case ch <- append([]byte{}, sc.scanner.Bytes()...):
				continue
			default:
				if discard {
					continue
				}
				ch <- append([]byte{}, sc.scanner.Bytes()...)
			}
		}
		sc.Err = sc.scanner.Err()
		close(ch)
	}()
	return ch
}
