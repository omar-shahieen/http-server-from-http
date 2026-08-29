package response

import (
	"fmt"
	"github/omar-shahieen/http-server/internal/headers"
	"io"
	"strconv"
)

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

// writerState tracks where the caller is in the required
// status-line -> headers -> body sequence.
type writerState int

const (
	writerStateStatusLine writerState = iota
	writerStateHeaders
	writerStateBody
)

// Writer wraps an io.Writer (typically a net.Conn) and enforces that
// callers write a status line, then headers, then the body, in that
// strict order, returning a clear error if they try to skip a step.
type Writer struct {
	writer io.Writer
	state  writerState
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer: w,
		state:  writerStateStatusLine,
	}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.state != writerStateStatusLine {
		return fmt.Errorf("cannot write status line: expected state %d, got %d", writerStateStatusLine, w.state)
	}

	var reasonPhrase string
	switch statusCode {
	case StatusOK:
		reasonPhrase = "OK"
	case StatusBadRequest:
		reasonPhrase = "Bad Request"
	case StatusInternalServerError:
		reasonPhrase = "Internal Server Error"
	default:
		reasonPhrase = ""
	}

	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, reasonPhrase)
	if _, err := w.writer.Write([]byte(statusLine)); err != nil {
		return err
	}

	w.state = writerStateHeaders
	return nil
}

func (w *Writer) WriteHeaders(h headers.Headers) error {
	if w.state != writerStateHeaders {
		return fmt.Errorf("cannot write headers: expected state %d, got %d", writerStateHeaders, w.state)
	}

	for key, value := range h {
		line := fmt.Sprintf("%s: %s\r\n", key, value)
		if _, err := w.writer.Write([]byte(line)); err != nil {
			return err
		}
	}

	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return err
	}

	w.state = writerStateBody
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("cannot write body: expected state %d, got %d", writerStateBody, w.state)
	}
	return w.writer.Write(p)
}

// GetDefaultHeaders returns the set of headers we always want to include
// in a response, given the length of the body in bytes. Callers can
// override any of these (e.g. Content-Type) with Headers.Set before
// passing the map to WriteHeaders.
func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h.Set("Content-Length", strconv.Itoa(contentLen))
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")
	return h
}
