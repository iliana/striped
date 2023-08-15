package main

import (
	"io"
	"log"
	"net"
	"os"
	"sync"

	"golang.org/x/net/proxy"
)

func main() {
	if len(os.Args) < 4 {
		log.Fatal("usage: striped LISTEN_ADDR SOCKS5_ADDR UPSTREAM_ADDR")
	}

	listenAddr, err := net.ResolveTCPAddr("tcp", os.Args[1])
	if err != nil {
		log.Fatal("LISTEN_ADDR is invalid:", err)
	}

	listener, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		log.Fatal("error while trying to start:", err)
	}

	defer listener.Close()
	log.Printf("listening on :%s\n", os.Args[1])

	dialer, err := proxy.SOCKS5("tcp", os.Args[2], nil, proxy.Direct)
	if err != nil {
		log.Fatal("error while trying to create dialer:", err)
	}

	upstream := os.Args[3]

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println("error while trying to accept connection:", err)
		}
		go handleConnection(conn, dialer, upstream)
	}
}

func handleConnection(conn *net.TCPConn, dialer proxy.Dialer, upstream string) {
	c, err := dialer.Dial("tcp", upstream)
	if err != nil {
		log.Println("error while trying to dial upstream:", err)
		return
	}
	proxyConn := c.(*net.TCPConn)

	// https://stackoverflow.com/a/75418345
	defer conn.Close()
	defer proxyConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(conn, proxyConn)
		conn.CloseWrite()
	}()
	go func() {
		defer wg.Done()
		io.Copy(proxyConn, conn)
		proxyConn.CloseWrite()
	}()

	wg.Wait()
}
