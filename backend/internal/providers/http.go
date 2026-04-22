package providers

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

// sharedHTTP is the client all providers use to fetch upstream feeds. It has a
// small timeout and a DialContext that rejects private network addresses by
// default — users configure feed URLs through an admin API, and without this
// guard a crafted URL could probe internal services (169.254.169.254 for cloud
// metadata, localhost, RFC1918, etc.).
//
// Opt out with STATUS_ALLOW_PRIVATE_NETWORKS=true if you need to aggregate
// internal status pages on the same network as the server.
var sharedHTTP = newSharedHTTP()

func newSharedHTTP() *http.Client {
	allowPrivate := strings.EqualFold(os.Getenv("STATUS_ALLOW_PRIVATE_NETWORKS"), "true")
	d := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	if !allowPrivate {
		d.Control = blockPrivateAddrs
	}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext:           d.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          32,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// blockPrivateAddrs runs after DNS resolution, just before the socket is
// connected. address is host:port with host already a literal IP, so we can
// classify it without a second lookup (which would open a TOCTOU window).
func blockPrivateAddrs(network, address string, _ syscall.RawConn) error {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return fmt.Errorf("unsupported network %q", network)
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("unexpected non-IP address %q", host)
	}
	if isBlockedIP(ip) {
		return fmt.Errorf("connection to %s blocked (private/loopback/link-local)", ip)
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

// Exposed for tests that need a plain client without the SSRF guard.
