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
	clusterId       string
	keystoneUrl     string
	qbertUrl        string
	bindAddr        string
	fwdHostAndPort  string
	servicesCidr    string
	servicesNet     *net.IPNet
	listenPort      int
	ks              bouncer.Keystone
	username        string
	password        string
	qb              qbert.Client
	logger          = log.New(os.Stderr, "", log.LstdFlags)
)

func main() {
	var token string
	flag.StringVar(&bindAddr, "bind", "0.0.0.0", "bind address")
	flag.StringVar(&servicesCidr, "services-cidr", "10.21.0.0/16", "kubernetes services CIDR")
	flag.StringVar(&fwdHostAndPort, "fwdaddr", "127.0.0.1:3020", "forwarder service host and port")
	flag.StringVar(&token, "token", "",
		"optional initial keystone token")
	flag.IntVar(&listenPort, "port", 0,
		"listening port (default: dynamic port)")
	flag.DurationVar(&keystoneTimeout, "keystone-timeout", defaultKeystoneTimeout,
		"The `duration` to wait for a response from Keystone")
	flag.Parse()
	if flag.NArg() != 6 {
		usage()
		os.Exit(1)
	}
	var err error
	_, servicesNet, err = net.ParseCIDR(servicesCidr)
	if err != nil {
		logger.Fatal("invalid services-cidr:", err)
	}
	keystoneUrl = flag.Arg(0)
	projectId = flag.Arg(1)
	username = flag.Arg(2)
	password = flag.Arg(3)
	qbertUrl = flag.Arg(4)
	clusterId = flag.Arg(5)
	ks, err = keystone.New(keystoneUrl, keystoneTimeout)
	if err != nil {
		logger.Fatal("failed to initialize keystone: ", err)
	}
	qb = qbert.Client{
		Url:       qbertUrl,
		Keystone:  ks,
		Username:  username,
		Password:  password,
		ProjectId: projectId,
		ClusterId: clusterId,
		Token:     token,
	}
	serve()
}

func usage() {
	cmd := os.Args[0]
	msg := `Master-worker proxy.
Usage: %s [OPTIONS] keystone-url project-id username password qbert-url cluster-id
`
	fmt.Printf(msg, cmd)
	flag.PrintDefaults()
}

func serve() {
	addr := fmt.Sprintf("%s:%d", bindAddr, listenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("failed to listen: %s", err)
	}
	logger.Printf("listening on %s", listener.Addr().String())
	for {
		cnx, err := listener.Accept()
		if err != nil {
			logger.Printf("warning: failed to accept: %s", err)
			continue
		}
		go handleConnection(cnx)
	}
}

func handleConnection(cnx net.Conn) {
	cnxId := proxylib.RandomString(8)
	defer proxylib.CloseConnection(cnx, logger, cnxId, "inbound")
	logger.Printf("[%s] accepted local connection", cnxId)
	netAddr, err := proxylib.OriginalDestination(cnxId, &cnx)
	if err != nil {
		logger.Printf("[%s] failed to obtain original destination: %s", cnxId, err)
		return
	}
	logger.Printf("[%s] original destination: %s", cnxId, netAddr)
	components := strings.Split(netAddr, ":")
	if len(components) != 2 {
		logger.Printf("[%s] invalid destination: %s", cnxId, netAddr)
		return
	}
	ip := components[0]
	port := components[1]
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		logger.Printf("[%s] malformed ip address: %s", cnxId, ip)
		return
	}
	if servicesNet.Contains(ipAddr) {
		logger.Printf("[%s] ip address %s within services network, aborting...",
			cnxId, ip)
		return
	}
	var uuid string
	uuid, err = qb.NodeUuidForIp(cnxId, ip)
	if err != nil {
		logger.Printf("[%s] node lookup failed: %s", cnxId, err)
		return
	}
	logger.Printf("[%s] node uuid: %s", cnxId, uuid)
	tcpConn := cnx.(*net.TCPConn)
	forwarder.ProxyTo(cnxId, logger, fwdHostAndPort, tcpConn, uuid, port)
}
