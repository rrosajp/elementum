package scrape

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/elgatito/elementum/config"
)

var (
	dialer = &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 15 * time.Second,
		DualStack: true,
	}

	// InternalProxyURL holds parsed internal proxy url
	internalProxyURL, _ = url.Parse("http://127.0.0.1:65222")

	directTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext:     CustomDialContext,
	}
	directClient = &http.Client{
		Transport: directTransport,
		Timeout:   15 * time.Second,
	}

	proxyTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyURL(internalProxyURL),
	}
	proxyClient = &http.Client{
		Transport: proxyTransport,
		Timeout:   30 * time.Second,
	}
)

// Reload ...
func Reload() {
	if config.Get().ProxyURL == "" || !config.Get().ProxyUseHTTP {
		directTransport.Proxy = nil
	} else {
		proxyURL, _ := url.Parse(config.Get().ProxyURL)
		directTransport.Proxy = http.ProxyURL(proxyURL)

		log.Debugf("Setting up proxy for direct client: %s", config.Get().ProxyURL)
	}
}

// GetClient ...
func GetClient() *http.Client {
	if !config.Get().InternalProxyEnabled {
		return directClient
	}

	return proxyClient
}

// CustomDial ...
func CustomDial(network, addr string) (net.Conn, error) {
	if !config.Get().InternalDNSEnabled {
		return dialer.Dial(network, addr)
	}

	addrs := strings.Split(addr, ":")
	if len(addrs) == 2 && len(addrs[0]) > 2 && strings.Index(addrs[0], ".") > -1 {
		if ipTest := net.ParseIP(addrs[0]); ipTest == nil {
			if ip, err := resolve(addrs[0]); err == nil && len(ip) > 0 {
				addr = ip[0] + ":" + addrs[1]
			}
		}
	}

	return dialer.Dial(network, addr)
}

// CustomDialContext ...
func CustomDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if !config.Get().InternalDNSEnabled {
		return dialer.DialContext(ctx, network, addr)
	}

	addrs := strings.Split(addr, ":")
	if len(addrs) == 2 && len(addrs[0]) > 2 && strings.Index(addrs[0], ".") > -1 {
		if ipTest := net.ParseIP(addrs[0]); ipTest == nil {
			if ip, err := resolve(addrs[0]); err == nil && len(ip) > 0 {
				addr = ip[0] + ":" + addrs[1]
			}
		}
	}

	return dialer.DialContext(ctx, network, addr)
}
