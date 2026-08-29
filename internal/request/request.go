package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

const bufferSize = 8
const crlf ="\r\n"
type parserState int

const (
	stateInitialized parserState = iota
	stateDone
)

type Request struct {
	RequestLine RequestLine
	state       parserState
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize)
	readToIndex := 0
	req := &Request{state: stateInitialized}

	for req.state != stateDone {
		// If buffer is full, double its capacity
		if readToIndex == len(buf) {
			newBuf := make([]byte, len(buf)*2)
			copy(newBuf, buf)
			buf = newBuf
		}

		n, err := reader.Read(buf[readToIndex:])
		if n > 0 {
			readToIndex += n

			bytesConsumed, parseErr := req.parse(buf[:readToIndex])
			if parseErr != nil {
				return nil, parseErr
			}

			if bytesConsumed > 0 {
				// Shift remaining unparsed bytes to the beginning of the buffer
				copy(buf, buf[bytesConsumed:readToIndex])
				readToIndex -= bytesConsumed
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
	}

	if req.state != stateDone {
		return nil, fmt.Errorf("incomplete request: reader closed before request line was fully parsed")
	}

	return req, nil
}

func (r *Request) parse(data []byte) (int, error) {
	switch r.state {
	case stateInitialized:
		bytesConsumed, reqLine, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if bytesConsumed == 0 {
			return 0, nil
		}
		r.RequestLine = reqLine
		r.state = stateDone
		return bytesConsumed, nil
	case stateDone:
		return 0, errors.New("error: trying to read data in a done state")
	default:
		return 0, errors.New("error: unknown state")
	}
}

func parseRequestLine(data []byte) (int, RequestLine, error) {
	idx := bytes.Index(data, []byte(crlf))
	if idx == -1 {
		return 0, RequestLine{}, nil
	}

	// Line length includes CRLF delimiter
	consumed := idx + 2
	line := string(data[:idx])

	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return 0, RequestLine{}, fmt.Errorf("invalid request line parts count: expected 3, got %d", len(parts))
	}

	method := parts[0]
	target := parts[1]
	versionRaw := parts[2]

	if len(method) == 0 {
		return 0, RequestLine{}, errors.New("method cannot be empty")
	}
	for _, ch := range method {
		if !unicode.IsUpper(ch) || !unicode.IsLetter(ch) {
			return 0, RequestLine{}, fmt.Errorf("invalid method %q: must contain only uppercase letters", method)
		}
	}

	if len(target) == 0 {
		return 0, RequestLine{}, errors.New("request target cannot be empty")
	}

	if !strings.HasPrefix(versionRaw, "HTTP/") {
		return 0, RequestLine{}, fmt.Errorf("invalid HTTP version prefix: %s", versionRaw)
	}

	version := strings.TrimPrefix(versionRaw, "HTTP/")
	if version != "1.1" {
		return 0, RequestLine{}, fmt.Errorf("unsupported HTTP version: %s", version)
	}

	return consumed, RequestLine{
		Method:        method,
		RequestTarget: target,
		HttpVersion:   version,
	}, nil
}