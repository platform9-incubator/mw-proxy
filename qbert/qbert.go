package qbert

import (
	"bouncer"
	"encoding/json"
	"fmt"
	"github.com/platform9/proxylib/pkg/proxylib"
	"log"
	"net/http"
	"os"
	"sync"
)

type Client struct {
	Url       string
	Keystone  bouncer.Keystone
	Username  string
	Password  string
	ProjectId string
	ClusterId string
	Token     string

	mtx      sync.Mutex
	nodeUuids []string
	ipToUuid map[string]string
	nodeIndex uint16
}

type Node struct {
	Uuid        string
	ClusterUuid string
	Status      string
	Name        string
	PrimaryIp   string
}

var logger = log.New(os.Stderr, "", log.LstdFlags)

//------------------------------------------------------------------------------
// InvalidateCache sets nodeUuids and ipToUuid to nil
func (cl *Client) InvalidateCache() {
	cl.mtx.Lock()
	defer cl.mtx.Unlock()
	cl.nodeUuids = nil
	cl.ipToUuid = nil
}

//------------------------------------------------------------------------------

func (cl *Client) refreshToken(cnxId string) error {
	ktw, err := cl.Keystone.ProjectTokenFromCredentials(
		cl.Username, cl.Password, cl.ProjectId,
	)
	if err != nil {
		return fmt.Errorf("keystone request failed: %s", err)
	}
	cl.Token = ktw.TokenID
	logger.Printf("[%s] refreshed token: %s", cnxId, cl.Token)
	return nil
}

//------------------------------------------------------------------------------

// NodeUuidForIp returns the qbert node uuid that has the specified ip
// as its PrimaryIp address.
// A non-nil error is returned if an error occurs while refreshing the node
// cache using qbert and keystone APIs, or if no error occurs but the node
// is not found.
func (cl *Client) NodeUuidForIp(cnxId string, ip string) (string, error) {
	cl.mtx.Lock()
	defer cl.mtx.Unlock()
	if cl.ipToUuid != nil {
		uuid, ok := cl.ipToUuid[ip]
		if ok {
			return uuid, nil
		}
	}
	if err := cl.refreshNodes(cnxId); err != nil {
		return "", fmt.Errorf("refreshNodes() failed: %s", err)
	}
	uuid, ok := cl.ipToUuid[ip]
	if !ok {
		return "", fmt.Errorf("no node with ip %s found", ip)
	}
	return uuid, nil
}

//------------------------------------------------------------------------------

// RandomNodeUuid returns a "randomly" selected node uuid.
// The current implementation just uses a round-robin index counter.
func (cl *Client) RandomNodeUuid(cnxId string) (string, error) {
	cl.mtx.Lock()
	defer cl.mtx.Unlock()
	if cl.ipToUuid == nil {
		if err := cl.refreshNodes(cnxId); err != nil {
			return "", fmt.Errorf("refreshNodes() failed: %s", err)
		}
	}
	if len(cl.nodeUuids) == 0 {
		return "", fmt.Errorf("there are no nodes in the cache")
	}
	i := int(cl.nodeIndex) % len(cl.nodeUuids)
	cl.nodeIndex += 1
	return cl.nodeUuids[i], nil
}

//------------------------------------------------------------------------------

// refreshNodes updates the cl.ipToUuid cache, possibly making calls to Keystone
// and Qbert as neeeded. Must be called with cl.mtx locked
func (cl *Client) refreshNodes(cnxId string) error {
	tokenRefreshed := false
	if cl.Token == "" {
		if err := cl.refreshToken(cnxId); err != nil {
			return err
		}
		tokenRefreshed = true
	}
	url := cl.Url + "/v2/" + cl.ProjectId + "/nodes"
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %s", err)
	}
	httpReq.Header.Set("User-Agent", "mw-proxy")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Auth-Token", cl.Token)
	httpClient := http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send 1st qbert request: %s", err)
	}
	defer proxylib.CloseConnection(resp.Body, logger, cnxId, "1st response body")
	if resp.StatusCode == http.StatusUnauthorized {
		if tokenRefreshed {
			return fmt.Errorf("1st qbert responded with 401 despite token refresh")
		}
		if err := cl.refreshToken(cnxId); err != nil {
			return err
		}
		httpReq.Header.Set("X-Auth-Token", cl.Token)
		resp, err = httpClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("failed to send 2nd qbert request: %s", err)
		}
		defer proxylib.CloseConnection(resp.Body, logger, cnxId, "2nd response body")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbert failure status: %d", resp.StatusCode)
	}
	var nodes []Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return fmt.Errorf("decoding qbert response: %s", err)
	}
	logger.Printf("[%s] nodes: %v", cnxId, nodes)
	cl.nodeUuids = []string{}
	cl.ipToUuid = make(map[string]string)
	for _, node := range nodes {
		if node.ClusterUuid == cl.ClusterId {
			cl.nodeUuids = append(cl.nodeUuids, node.Uuid)
			cl.ipToUuid[node.PrimaryIp] = node.Uuid
		}
	}
	logger.Printf("[%s] node map: %v", cnxId, cl.ipToUuid)
	return nil
}
