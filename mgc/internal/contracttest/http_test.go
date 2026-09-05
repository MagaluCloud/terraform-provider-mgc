//go:build contract

package contracttest

import (
	"net/http"
	"testing"
)

func TestContractTransportRejectsUnregisteredHosts(t *testing.T) {
	guard := &localTransport{servers: make(map[string]http.RoundTripper)}
	req, err := http.NewRequest("GET", "https://unregistered.invalid/resource", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := guard.RoundTrip(req)
	if err == nil || response != nil || len(guard.blocked) != 1 {
		t.Fatalf("request must fail before reaching the network: response=%v err=%v blocked=%v", response, err, guard.blocked)
	}
}

func TestAliasRoutesToLocalTLSWithoutChangingOriginalRequest(t *testing.T) {
	local := Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object" || r.URL.RawQuery != "version=1" {
			t.Errorf("request path was changed: %s", r.URL)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	Alias(t, "https://fixture.invalid", local)
	req, err := http.NewRequest("GET", "https://fixture.invalid/object?version=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := network.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected response: %d", response.StatusCode)
	}
	if req.URL.Host != "fixture.invalid" {
		t.Fatal("routing mutated the SDK request")
	}
}
