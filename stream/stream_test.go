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
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestMockClosed(t *testing.T) {
	var err error
	var s = new(StreamClient)
	s.Close()
	err = s.Connect("test", "1234")
	if err == nil || err.Error() != "cannot connect closed client" {
		t.Errorf("unexpected: %s\n", err)
	}
	var r = s.Scan(10, false)
	if r != nil {
		t.Error("scan on closed client returned channel\n")
	}
	if s.Err.Error() != "cannot scan closed client" {
		t.Errorf("unexpected: %s\n", s.Err)
	}
	s.scanAndWatch(false)
	s.Close()
}

func TestMockValid(t *testing.T) {
	var err error
	var srv, clt net.Conn
	var s = new(StreamClient)
	srv, clt = net.Pipe()
	s.conn = clt
	var ch = s.Scan(5, false)
	go func(w io.WriteCloser) {
		var e error
		var f *os.File
		f, e = os.Open("data.json")
		if e != nil {
			t.Fatal(err)
		}
		defer f.Close()
		_, e = io.Copy(w, f)
		if e != nil {
			t.Fatal(err)
		}
		w.Close()
	}(srv)
	for {
		var msg []byte
		var ok bool
		msg, ok = <-ch
		if !ok {
			if s.Err != nil {
				fmt.Printf("ERR: %s\n\n", s.Err)
			}
			break
		}
		fmt.Printf("Len: %d\n", len(msg))
		//fmt.Printf("Len: %d\nBegin: %s\nEnd: %s\n", len(msg), msg[:10], msg[len(msg)-10:])
		//fmt.Printf("Queue: %d of %d\n\n", len(ch), cap(ch))
	}
	s.Close()
}

func TestMockOverflow(t *testing.T) {
	testOverflow(t, false)
	testOverflow(t, true)
}

func testOverflow(t *testing.T, wait bool) {
	var srv, clt net.Conn
	var s = new(StreamClient)
	srv, clt = net.Pipe()
	s.conn = clt
	var ch = s.Scan(5, wait)
	go func(w io.WriteCloser) {
		var e error
		var b []byte
		for i := 0; i < 10; i++ {
			b = []byte(`{"acList":[{"Icao":"aabbcc"}]}`)
			_, e = w.Write(b)
			if e != nil {
				fmt.Printf("write error: %s\n", e)
				break
			}
		}
		w.Close()
	}(srv)
	for len(ch) < cap(ch) {
		time.Sleep(100 * time.Millisecond) // wait for channel to fill
	}
	for {
		var msg []byte
		var ok bool
		msg, ok = <-ch
		if !ok {
			if s.Err != nil {
				fmt.Printf("ERR: %s\n\n", s.Err)
			}
			break
		}
		fmt.Printf("Len: %d\n", len(msg))
		// fmt.Printf("Len: %d\nBegin: %s\nEnd: %s\n", len(msg), msg[:10], msg[len(msg)-10:])
		// fmt.Printf("Queue: %d of %d\n\n", len(ch), cap(ch))
		s.Close() // test close while receiving
	}
	s.Close()
}

func TestMockEOF(t *testing.T) {
	var srv, clt net.Conn
	var s = new(StreamClient)
	srv, clt = net.Pipe()
	s.conn = clt
	var ch = s.Scan(5, false)
	go func(w io.WriteCloser) {
		var e error
		var b = []byte(`{"acList":[{"Icao":"aabbcc"}]}{"acList":`)
		_, e = w.Write(b)
		if e != nil {
			fmt.Printf("write error: %s\n", e)
		}
		w.Close()
	}(srv)
	for {
		var msg []byte
		var ok bool
		msg, ok = <-ch
		if !ok {
			if s.Err != nil {
				fmt.Printf("ERR: %s\n\n", s.Err)
			}
			break
		}
		fmt.Printf("Len: %d\n", len(msg))
		// fmt.Printf("Len: %d\nBegin: %s\nEnd: %s\n", len(msg), msg[:10], msg[len(msg)-10:])
		// fmt.Printf("Queue: %d of %d\n\n", len(ch), cap(ch))
	}
	s.Close()
}

func TestLive(t *testing.T) {
	var err error

	rand.Seed(time.Now().UnixNano())
	var port = strconv.Itoa(rand.Intn(40000) + 10000)
	testConnect(t, port, true, 0)

	var l net.Listener
	l, err = net.Listen("tcp", net.JoinHostPort("localhost", port))
	if err != nil {
		t.Fatalf("listener error: %s\n", err)
	}

	go testConnect(t, port, false, 0)

	var c net.Conn
	c, err = l.Accept()
	if err != nil {
		t.Fatalf("accept error: %s\n", err)
	}
	c.Close()

	go testConnect(t, port, false, 2)

	c, err = l.Accept()
	if err != nil {
		t.Fatalf("accept error: %s\n", err)
	}
	c.Close()

	c, err = l.Accept()
	if err != nil {
		t.Fatalf("accept error: %s\n", err)
	}
	l.Close()
	c.Close()

	fmt.Println("wait for reconnect timeout")
	time.Sleep(5 * time.Second)
}

func testConnect(t *testing.T, port string, fail bool, reconnect int) {
	var err error

	var s = new(StreamClient)
	s.ReconnectMax = reconnect
	err = s.Connect("localhost", port)
	if fail {
		if err == nil {
			t.Error("expected error, got nil\n")
		}
		return
	}
	if err != nil {
		t.Fatalf("connect error: %s\n", err)
	}

	var ch = s.Scan(5, false)

	for i := 0; i < 5; i++ {
		var msg []byte
		var ok bool
		msg, ok = <-ch
		if !ok {
			if s.Err != nil {
				fmt.Printf("ERR: %s\n\n", s.Err)
			}
			break
		}
		fmt.Printf("Len: %d\n", len(msg))
	}
	s.Close()
}
