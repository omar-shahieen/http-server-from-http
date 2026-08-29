package server

import (
	"fmt"
	"github/omar-shahieen/http-server/internal/request"
	"github/omar-shahieen/http-server/internal/response"
	"log"
	"net"
	"strconv"
	"sync/atomic"
)

// Handler is the function signature user code implements to handle a
// request. It has full control over the response via w: it must write
// a status line, then headers, then a body, using w's methods, even in
// error cases.
type Handler func(w *response.Writer, req *request.Request)

// Server holds the state of a running HTTP server.
type Server struct {
	listener net.Listener
	handler  Handler
	closed   atomic.Bool
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, fmt.Errorf("could not start listener: %w", err)
	}

	s := &Server{
		listener: listener,
		handler:  handler,
	}

	go s.listen()

	return s, nil
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			log.Printf("error accepting connection: %v", err)
			continue
		}

		go s.handle(conn)
	}
}

// handle parses the incoming request and hands off full control of the
// response to the server's handler. If the request itself fails to
// parse, we write a generic 400 ourselves, since there's no valid
// *request.Request to hand the user's handler.
func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	req, err := request.RequestFromReader(conn)
	if err != nil {
		w := response.NewWriter(conn)
		body := []byte(err.Error() + "\n")

		if writeErr := w.WriteStatusLine(response.StatusBadRequest); writeErr != nil {
			log.Printf("error writing status line: %v", writeErr)
			return
		}
		h := response.GetDefaultHeaders(len(body))
		if writeErr := w.WriteHeaders(h); writeErr != nil {
			log.Printf("error writing headers: %v", writeErr)
			return
		}
		if _, writeErr := w.WriteBody(body); writeErr != nil {
			log.Printf("error writing body: %v", writeErr)
		}
		return
	}

	w := response.NewWriter(conn)
	s.handler(w, req)
}
