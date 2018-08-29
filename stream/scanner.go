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
	"bufio"
	"bytes"
	"errors"
	"io"
)

// scanner is a customized bufio.Scanner.
type scanner struct {
	*bufio.Scanner

	stop bool

	inMsg bool
}

// newStreamScaner returns a scanner capable of splitting a VRS stream
// into individual JSON messages.
func newStreamScanner(r io.Reader, maxBuf int) *scanner {
	var s = new(scanner)
	s.Scanner = bufio.NewScanner(r)
	if maxBuf > 0 {
		var b = make([]byte, 0, maxBuf)
		s.Buffer(b, maxBuf)
	}
	s.Split(s.splitStream)
	return s
}

// splitStream implements the bufio.SplitFunc interface.
func (s *scanner) splitStream(data []byte, eof bool) (a int, b []byte, c error) {
	// find the first instance of each JSON token
	var msgEnd = bytes.Index(data, []byte(`}]}`))
	var msgBegin = bytes.Index(data, []byte(`{"ac`))

	// whole message in buffer
	if s.inMsg && msgEnd != -1 {
		s.inMsg = false
		return msgEnd + 3, data[:msgEnd+3], nil
	}
	// partial message found
	if s.inMsg && msgEnd == -1 {
		// not EOF, request more data
		if !eof {
			return 0, nil, nil
		}
		// EOF, return error
		return 0, nil, errors.New("premature EOF")
	}

	// stop requested
	if s.stop {
		return 0, nil, errors.New("stop requested")
	}

	// no begin found
	if msgBegin == -1 {
		// not EOF, request more data
		if !eof {
			return 0, nil, nil
		}
		// EOF, return error
		return 0, nil, errors.New("premature EOF")
	}
	// begin found
	s.inMsg = true
	// no end found
	if msgEnd == -1 {
		// not EOF, request more data
		if !eof {
			return 0, nil, nil
		}
		// EOF, return error
		return 0, nil, errors.New("premature EOF")
	}
	// end found
	if msgEnd != -1 {
		s.inMsg = false
		return msgEnd + 3, data[:msgEnd+3], nil
	}
	// should not reach here
	return 0, nil, errors.New("splitter error")
}
