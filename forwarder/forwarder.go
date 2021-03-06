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
	destHost string,
) {
	req, err := http.NewRequest(
		"GET",
		"http://" + fwdHostAndPort,
		nil,
	)
	if err != nil {
		logger.Printf("[%s] failed to create request: %s", sid, err.Error())
		return
	}
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "platform9.com/forwarder.dynamic.v2")
	req.Header.Set("hostid", hostId)
	req.Header.Set("hostlocalport", hostPort)
	if destHost != "" {
		req.Header.Set("destinationhost", destHost)
	}

	cnx, err := net.Dial("tcp", fwdHostAndPort)
	if err != nil {
		logger.Printf("[%s] failed to dial: %s", sid, err.Error())
		return
	}
	defer proxylib.CloseConnection(cnx, logger, sid, "outbound")
	netConn := cnx.(*net.TCPConn)
	if err := req.Write(netConn); err != nil {
		logger.Printf("[%s] failed to send upgrade request: %s",
			sid, err.Error())
		return
	}
	br := bufio.NewReaderSize(netConn, 8192)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		logger.Printf("[%s] failed to read upgrade response: %s",
			sid, err.Error())
		return
	}
	defer proxylib.CloseConnection(resp.Body, logger, sid, "upgrade response")
	if resp.StatusCode != http.StatusSwitchingProtocols {
		logger.Printf("[%s] unexpected upgrade status code: %d",
			sid, resp.StatusCode)
		return
	}
	logger.Printf("[%s] connection established", sid)
	proxylib.FerryBytes(conn, netConn, sid, 0)
}
