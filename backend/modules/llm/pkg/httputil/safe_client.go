// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// maxImageFetchBytes caps the size of a server-side image fetch to avoid memory
// exhaustion from a malicious/huge response.
const maxImageFetchBytes = 20 << 20 // 20 MiB

// safeHTTPClient is an *http.Client whose dialer refuses to connect to private,
// loopback, link-local or cloud-metadata addresses. The check runs at dial time
// against the resolved IP, which also defeats DNS rebinding. Use it for any
// server-side fetch of a user/model-supplied URL (SSRF defense).
var safeHTTPClient = newSafeHTTPClient(15 * time.Second)

func newSafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return fmt.Errorf("blocked: unresolved host %q", host)
			}
			if isBlockedIP(ip) {
				return fmt.Errorf("blocked: connection to non-public address %s is not allowed", ip)
			}
			return nil
		},
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("blocked: too many redirects")
			}
			return nil
		},
	}
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.Equal(net.ParseIP("169.254.169.254")) { // cloud metadata
		return true
	}
	return false
}

// validateFetchURL ensures the URL uses an http(s) scheme before it is fetched.
func validateFetchURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("blocked: unsupported url scheme %q", u.Scheme)
	}
	return nil
}
