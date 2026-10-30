// Loopback-only concurrent TLS 1.3 camouflage handshake fixture for REALITY.
package main

import (
	"crypto/tls"
	"io"
	"net"
	"os"
	"time"
)

func main() {
	cert, err := tls.LoadX509KeyPair(os.Args[2], os.Args[3])
	if err != nil {
		panic(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:"+os.Args[1])
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	config := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13, NextProtos: []string{"h2", "http/1.1"}}
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go func() {
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(15 * time.Second))
			secured := tls.Server(conn, config)
			if secured.Handshake() == nil {
				io.Copy(io.Discard, secured)
			}
		}()
	}
}
