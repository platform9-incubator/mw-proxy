package main

import (
	"bouncer"
	"bouncer/keystone"
	"flag"
	"fmt"
	"github.com/platform9-incubator/mw-proxy/forwarder"
	"github.com/platform9-incubator/mw-proxy/qbert"
	"github.com/platform9/proxylib/pkg/proxylib"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	defaultKeystoneTimeout = time.Duration(30) * time.Second
)

var (
	keystoneTimeout time.Duration
	projectId       string
	keystoneUrl     string
	qbertUrl        string
	bindAddr        string
	fwdHostAndPort  string
	listenPort      int
	ks              bouncer.Keystone
	username        string
	password        string
	qb              qbert.Client
	logger          = log.New(os.Stderr, "mw-proxy ", log.LstdFlags)
)

func main() {
	var token string
	flag.StringVar(&bindAddr, "bind", "0.0.0.0", "bind address")
	flag.StringVar(&fwdHostAndPort, "fwdaddr", "127.0.0.1:3020", "forwarder service host and port")
	flag.StringVar(&token, "token", "",
		"optional initial keystone token")
	flag.IntVar(&listenPort, "port", 0,
		"listening port (default: dynamic port)")
	flag.DurationVar(&keystoneTimeout, "keystone-timeout", defaultKeystoneTimeout,
		"The `duration` to wait for a response from Keystone")
	flag.Parse()
	if flag.NArg() != 5 {
		usage()
		os.Exit(1)
	}
	keystoneUrl = flag.Arg(0)
	projectId = flag.Arg(1)
	username = flag.Arg(2)
	password = flag.Arg(3)
	qbertUrl = flag.Arg(4)
	var err error
	ks, err = keystone.New(keystoneUrl, keystoneTimeout)
	if err != nil {
		log.Fatal("failed to initialize keystone: ", err)
	}
	qb = qbert.Client{
		Url:       qbertUrl,
		Keystone:  ks,
		Username:  username,
		Password:  password,
		ProjectId: projectId,
		Token:     token,
	}
	serve()
}

func usage() {
	cmd := os.Args[0]
	msg := `Master-worker proxy.
Usage: %s [OPTIONS] keystone-url project-id username password qbert-url
`
	fmt.Printf(msg, cmd)
	flag.PrintDefaults()
}

func serve() {
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
	defer proxylib.CloseConnection(cnx, logger, cnxId, "inbound")
	log.Printf("[%s] accepted local connection", cnxId)
	netAddr, err := proxylib.OriginalDestination(&cnx)
	if err != nil {
		log.Printf("[%s] failed to obtain original destination: %s", cnxId, err)
		return
	}
	log.Printf("[%s] original destination: %s", cnxId, netAddr)
	components := strings.Split(netAddr, ":")
	if len(components) != 2 {
		logger.Printf("[%s] invalid destination: %s", cnxId, netAddr)
		return
	}
	ip := components[0]
	port := components[1]
	var uuid string
	uuid, err = qb.NodeUuidForIp(ip)
	if err != nil {
		logger.Printf("[%s] node lookup failed: %s", cnxId, err)
		return
	}
	tcpConn := cnx.(*net.TCPConn)
	forwarder.ProxyTo(cnxId, logger, fwdHostAndPort, tcpConn, uuid, port)
}
