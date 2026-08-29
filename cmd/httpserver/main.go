package main

import (
	"github/omar-shahieen/http-server/internal/request"
	"github/omar-shahieen/http-server/internal/response"
	"github/omar-shahieen/http-server/internal/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const port = 42069

const badRequestBody = `<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>
`

const internalServerErrorBody = `<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>
`

const okBody = `<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>
`

func handler(w *response.Writer, req *request.Request) {
	var statusCode response.StatusCode
	var body []byte

	switch req.RequestLine.RequestTarget {
	case "/yourproblem":
		statusCode = response.StatusBadRequest
		body = []byte(badRequestBody)
	case "/myproblem":
		statusCode = response.StatusInternalServerError
		body = []byte(internalServerErrorBody)
	default:
		statusCode = response.StatusOK
		body = []byte(okBody)
	}

	if err := w.WriteStatusLine(statusCode); err != nil {
		log.Printf("error writing status line: %v", err)
		return
	}

	headers := response.GetDefaultHeaders(len(body))
	headers.Set("Content-Type", "text/html")
	if err := w.WriteHeaders(headers); err != nil {
		log.Printf("error writing headers: %v", err)
		return
	}

	if _, err := w.WriteBody(body); err != nil {
		log.Printf("error writing body: %v", err)
	}
}

func main() {
	server, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
