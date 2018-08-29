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
	"testing"
)

func TestStream(t *testing.T) {
	var err error

	var s = new(StreamClient)
	s.ReconnectMax = 3
	err = s.Connect("localhost", "33099")
	if err != nil {
		t.Fatal(err)
	}
	var ch = s.Scan(5, false)

	for i := 0; i < 100; i++ {
		var msg []byte
		var ok bool
		msg, ok = <-ch
		if !ok {
			fmt.Printf("ERR: %s\n\n", s.Err)
			break
		}
		fmt.Printf("Len: %d\nBegin: %s\nEnd: %s\n", len(msg), msg[:10], msg[len(msg)-10:])
		fmt.Printf("Queue: %d of %d\n\n", len(ch), cap(ch))
	}
	s.Close()
}
