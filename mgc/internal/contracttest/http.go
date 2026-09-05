//go:build contract

package contracttest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

type localTransport struct {
	mu      sync.RWMutex
	servers map[string]http.RoundTripper
	blocked []string
}

var network = &localTransport{servers: make(map[string]http.RoundTripper)}

func (g *localTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	g.mu.Lock()
	transport := g.servers[r.URL.Scheme+"://"+r.URL.Host]
	if transport == nil {
		g.blocked = append(g.blocked, r.URL.Host)
	}
	g.mu.Unlock()
	if transport == nil {
		return nil, fmt.Errorf("contract test blocked unregistered endpoint %s", r.URL.Host)
	}
	return transport.RoundTrip(r)
}

// Server registers a TLS fixture and trusts only its generated test certificate.
func Server(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	s := httptest.NewTLSServer(handler)
	u, err := url.Parse(s.URL)
	if err != nil {
		s.Close()
		t.Fatalf("parse local TLS fixture URL %q: %v", s.URL, err)
	}
	transport := s.Client().Transport.(*http.Transport).Clone()
	transport.Proxy = nil
	network.mu.Lock()
	network.servers[u.Scheme+"://"+u.Host] = transport
	network.mu.Unlock()
	t.Cleanup(func() {
		s.Close()
		transport.CloseIdleConnections()
		network.mu.Lock()
		delete(network.servers, u.Scheme+"://"+u.Host)
		network.mu.Unlock()
	})
	return s
}

type redirectedTransport struct {
	endpoint *url.URL
	next     http.RoundTripper
}

func (r redirectedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	local := req.Clone(req.Context())
	local.URL.Scheme = r.endpoint.Scheme
	local.URL.Host = r.endpoint.Host
	local.Host = r.endpoint.Host
	return r.next.RoundTrip(local)
}

// Alias sends an SDK's fixed destination to an already registered local server.
// Rewriting happens before DNS and transport; remote network access stays blocked.
func Alias(t *testing.T, origin string, local *httptest.Server) {
	t.Helper()
	remote, err := url.Parse(origin)
	if err != nil || remote.Scheme != "https" || remote.Host == "" || remote.Path != "" {
		t.Fatalf("invalid HTTPS origin %q", origin)
	}
	endpoint, err := url.Parse(local.URL)
	if err != nil {
		t.Fatal(err)
	}
	network.mu.Lock()
	next, ok := network.servers[local.URL]
	if !ok {
		network.mu.Unlock()
		t.Fatal("alias target must be a registered local server")
	}
	if _, exists := network.servers[origin]; exists {
		network.mu.Unlock()
		t.Fatalf("duplicate API alias %q", origin)
	}
	network.servers[origin] = redirectedTransport{endpoint: endpoint, next: next}
	network.mu.Unlock()
	t.Cleanup(func() { network.mu.Lock(); delete(network.servers, origin); network.mu.Unlock() })
}
