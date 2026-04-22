package providers

import (
	"net"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.5", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true}, // cloud metadata
		{"fe80::1", true},
		{"fc00::1", true},
		{"0.0.0.0", true},
		{"224.0.0.1", true},
		{"8.8.8.8", false},
		{"93.184.216.34", false}, // example.com
		{"2606:2800:220:1::", false},
	}
	for _, tc := range cases {
		t.Run(tc.ip, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("bad test: %q is not a valid IP", tc.ip)
			}
			if got := isBlockedIP(ip); got != tc.blocked {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tc.ip, got, tc.blocked)
			}
		})
	}
}
