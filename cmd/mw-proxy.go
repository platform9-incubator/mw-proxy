package main

import (
	"flag"
	"fmt"
	"github.com/platform9/proxylib/pkg/proxylib"
	"log"
	"net"
)

func main() {
	var bindAddr string
	var lstnPort int
	flag.StringVar(&bindAddr, "bind", "0.0.0.0",
		"bind address")
	flag.IntVar(&lstnPort, "port", 0,
		"listening port (default: dynamic port)")
	flag.Parse()
	serve(bindAddr, lstnPort)
}

func serve(
	bindAddr string,
	listenPort int,
) {
	addr := fmt.Sprintf("%s:%d", bindAddr, listenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}
	log.Printf("listening on %s", listener.Addr().String())
	for {
		cnx, err := listener.Accept()
		if err != nil {
			log.Printf("warning: failed to accept: %s", err)
			continue
		}
		go handleConnection(cnx)
	}
}

func handleConnection(
	cnx net.Conn,
) {
	cnxId := proxylib.RandomString(8)
	defer proxylib.CloseConnection(cnx, cnxId, "inbound")
	log.Printf("[%s] accepted local connection", cnxId)
	netAddr, err := proxylib.RealServerAddress(&cnx)
	if err != nil {
		log.Printf("[%s] failed to obtain real address: %s", cnxId, err)
		return
	}
	log.Printf("[%s] real address: %s", cnxId, netAddr)
}