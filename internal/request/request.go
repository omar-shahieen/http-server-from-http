package request

import (
	"bytes"
	"errors"
	"fmt"
	"github/omar-shahieen/http-server/internal/headers"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const bufferSize = 8
const crlf = "\r\n"

type parserState int

const (
	stateInitialized parserState = iota
	stateParsingHeaders
	stateParsingBody
	stateDone
)

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
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
	req := &Request{
		state:   stateInitialized,
		Headers: headers.NewHeaders(),
		Body:    make([]byte, 0),
	}

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
		return nil, fmt.Errorf("incomplete request: reader closed before request was fully parsed")
	}

	return req, nil
}

// parse repeatedly calls parseSingle against data until either the state
// machine reaches stateDone, or a single pass can't make any more progress
// (meaning we need more data from the reader before we can continue).
func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0

	for r.state != stateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return 0, err
		}
		if n == 0 {
			// Not enough data to make further progress this pass;
			// wait for more bytes from the reader.
			break
		}
		totalBytesParsed += n
	}

	return totalBytesParsed, nil
}

// parseSingle handles a single state transition / single parse operation.
func (r *Request) parseSingle(data []byte) (int, error) {
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
		r.state = stateParsingHeaders
		return bytesConsumed, nil

	case stateParsingHeaders:
		bytesConsumed, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if bytesConsumed == 0 {
			return 0, nil
		}
		if done {
			r.state = stateParsingBody
		}
		return bytesConsumed, nil

	case stateParsingBody:
		contentLengthStr := r.Headers.Get("Content-Length")
		if contentLengthStr == "" {
			// No Content-Length header: assume no body, we're done.
			r.state = stateDone
			return 0, nil
		}

		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			return 0, fmt.Errorf("invalid Content-Length header: %q", contentLengthStr)
		}

		r.Body = append(r.Body, data...)

		if len(r.Body) > contentLength {
			return 0, fmt.Errorf("body length %d exceeds reported Content-Length %d", len(r.Body), contentLength)
		}

		if len(r.Body) == contentLength {
			r.state = stateDone
		}

		return len(data), nil

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
