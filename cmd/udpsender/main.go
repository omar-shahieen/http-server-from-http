package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	// Resolve the UDP address
	addr, err := net.ResolveUDPAddr("udp", "localhost:42069")
	if err != nil {
		log.Fatalf("failed to resolve UDP address: %v", err)
	}

	// Prepare the UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("failed to dial UDP: %v", err)
	}
	defer conn.Close()

	// Create a buffered reader for standard input
	reader := bufio.NewReader(os.Stdin)

	// Infinite loop to prompt and send messages
	for {
		fmt.Print("> ")

		// Read input until newline
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("error reading input: %v", err)
			continue
		}

		// Write the line to the UDP connection
		_, err = conn.Write([]byte(line))
		if err != nil {
			log.Printf("error sending UDP packet: %v", err)
			continue
		}
	}
}