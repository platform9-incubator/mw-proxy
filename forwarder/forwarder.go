package forwarder

import (
	"bufio"
	"github.com/platform9/proxylib/pkg/proxylib"
	"net"
	"net/http"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

// ProxyTo proxies the specified connection to the specified destination
// host and port using the Platform9 Dynamic Forwarder Protocol V2 API
func ProxyTo(
	sid string,
	logger Logger,
	fwdHostAndPort string,
	conn *net.TCPConn,
	hostId,
	hostPort string,
) {
	req, err := http.NewRequest(
		"GET",
		"http://"+fwdHostAndPort,
		nil,
	)
	if err != nil {
		logger.Printf("%s failed to create request: %s", sid, err.Error())
		return
	}
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "platform9.com/forwarder.dynamic.v2")
	req.Header.Set("hostid", hostId)
	req.Header.Set("hostlocalport", hostPort)

	cnx, err := net.Dial("tcp", fwdHostAndPort)
	if err != nil {
		logger.Printf("%s failed to dial: %s", sid, err.Error())
		return
	}
	netConn := cnx.(*net.TCPConn)
	defer proxylib.CloseConnection(netConn, logger, sid, "outgoing")
	if err := req.Write(netConn); err != nil {
		logger.Printf("%s failed to write request: %s", sid, err.Error())
		return
	}
	br := bufio.NewReaderSize(netConn, 8192)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		logger.Printf("%s failed to read response: %s", sid, err.Error())
		return
	}
	if resp.StatusCode != 101 {
		logger.Printf("%s unexpected status code: %d", sid, resp.StatusCode)
		return
	}
	logger.Printf("%s connection established", sid)
	proxylib.FerryBytes(conn, netConn, sid, 0)
}
