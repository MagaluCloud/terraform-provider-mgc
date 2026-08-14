package acctest

import (
	"io"
	"math/rand/v2"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func newReq(t *testing.T, method, url, body string) *http.Request {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	return req
}

func TestMatcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  *http.Request
		rec  cassette.Request
		want bool
	}{
		{
			name: "identical get",
			req:  newReq(t, http.MethodGet, "https://api.example.com/v1/clusters", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://api.example.com/v1/clusters", Body: ""},
			want: true,
		},
		{
			name: "different host still matches (path only)",
			req:  newReq(t, http.MethodGet, "https://replay.invalid/v1/clusters", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://api.example.com/v1/clusters", Body: ""},
			want: true,
		},
		{
			// Distinct UUIDs must NOT collapse: normalizing them erased the only
			// field telling sibling reads apart (subnet a vs b), causing go-vcr
			// to swap their responses and drift the state. Guards the removal.
			name: "distinct uuids in path do not match",
			req:  newReq(t, http.MethodGet, "https://h/v1/clusters/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://h/v1/clusters/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Body: ""},
			want: false,
		},
		{
			// The same UUID still matches: a resource id flows from the recorded
			// create response, so reads of that resource replay cleanly.
			name: "same uuid in path matches",
			req:  newReq(t, http.MethodGet, "https://h/v1/clusters/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://other/v1/clusters/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Body: ""},
			want: true,
		},
		{
			name: "region path segment is normalized",
			req:  newReq(t, http.MethodGet, "https://h/kubernetes/v1/clusters", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://api.magalu.cloud/br-se1/kubernetes/v1/clusters", Body: ""},
			want: true,
		},
		{
			name: "different regions still match",
			req:  newReq(t, http.MethodGet, "https://h/br-ne1/kubernetes/v1/clusters", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://h/br-se1/kubernetes/v1/clusters", Body: ""},
			want: true,
		},
		{
			name: "reordered json body matches",
			req:  newReq(t, http.MethodPost, "https://h/v1/clusters", `{"a":1,"name":"x"}`),
			rec:  cassette.Request{Method: "POST", URL: "https://h/v1/clusters", Body: `{"name":"x","a":1}`},
			want: true,
		},
		{
			name: "reordered query matches",
			req:  newReq(t, http.MethodGet, "https://h/v1/clusters?b=2&a=1", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://h/v1/clusters?a=1&b=2", Body: ""},
			want: true,
		},
		{
			name: "different method does not match",
			req:  newReq(t, http.MethodDelete, "https://h/v1/clusters/x", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://h/v1/clusters/x", Body: ""},
			want: false,
		},
		{
			name: "different body does not match",
			req:  newReq(t, http.MethodPost, "https://h/v1/clusters", `{"name":"a"}`),
			rec:  cassette.Request{Method: "POST", URL: "https://h/v1/clusters", Body: `{"name":"b"}`},
			want: false,
		},
		{
			name: "different path does not match",
			req:  newReq(t, http.MethodGet, "https://h/v1/nodepools", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://h/v1/clusters", Body: ""},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := matcher(tt.req, tt.rec); got != tt.want {
				t.Fatalf("matcher = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLoopbackEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		raw  string
		want bool
	}{
		{"http://localhost:8080", true},
		{"http://127.0.0.1:8080", true},
		{"https://127.0.0.1", true},
		{"http://[::1]:8080", true},
		{"http://127.5.5.5:9000", true},
		{"", false},
		{"https://api.magalu.cloud", false},
		{"https://br-se1.magaluobjects.com", false},
		{"not a url", false},
	}
	for _, tt := range tests {
		if got := isLoopbackEndpoint(tt.raw); got != tt.want {
			t.Errorf("isLoopbackEndpoint(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestParseVCRMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw     string
		want    recorder.Mode
		wantErr bool
	}{
		{raw: "", want: recorder.ModeReplayOnly},
		{raw: "replay", want: recorder.ModeReplayOnly},
		{raw: " Record ", want: recorder.ModeRecordOnly},
		{raw: "off", want: recorder.ModePassthrough},
		{raw: "live", want: recorder.ModePassthrough},
		// Recording must always be explicit: the old auto mode is rejected.
		{raw: "auto", wantErr: true},
		{raw: "bogus", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseVCRMode(tt.raw)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseVCRMode(%q): expected error, got mode %v", tt.raw, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseVCRMode(%q): %v", tt.raw, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseVCRMode(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestMatcherRestoresBodyAcrossCalls(t *testing.T) {
	t.Parallel()
	// The recorder calls the matcher once per candidate interaction, so the body
	// must survive a non-matching first call and still match on a second call.
	req := newReq(t, http.MethodPost, "https://h/v1/clusters", `{"name":"x"}`)

	miss := cassette.Request{Method: "POST", URL: "https://h/v1/clusters", Body: `{"name":"other"}`}
	hit := cassette.Request{Method: "POST", URL: "https://h/v1/clusters", Body: `{"name":"x"}`}

	if matcher(req, miss) {
		t.Fatal("first (non-matching) call should not match")
	}
	if !matcher(req, hit) {
		t.Fatal("second call should match: body was not restored after the first call")
	}
}

func TestScrubHook(t *testing.T) {
	t.Parallel()

	i := &cassette.Interaction{
		Request: cassette.Request{
			Headers: http.Header{},
			Body:    `{"name":"x","api_key":"super-secret","nested":{"key_pair_secret":"also-secret"}}`,
		},
		Response: cassette.Response{
			Headers: http.Header{},
			Body:    `{"id":"abc","token":"leaky"}`,
		},
	}
	i.Request.Headers.Set("Authorization", "Bearer xyz")
	i.Request.Headers.Set("X-API-Key", "00000000-0000-4000-8000-000000000000")
	i.Request.Headers.Set("User-Agent", "MgcTF/test")
	i.Response.Headers.Set("Date", "now")
	i.Response.Headers.Set("X-Request-Id", "req-123")
	i.Response.Headers.Set("Content-Type", "application/json")

	if err := scrubHook(i); err != nil {
		t.Fatalf("scrubHook: %v", err)
	}

	if got := i.Request.Headers.Get("Authorization"); got != "" {
		t.Errorf("Authorization not scrubbed: %q", got)
	}
	if got := i.Request.Headers.Get("X-Api-Key"); got != "" {
		t.Errorf("X-Api-Key not scrubbed: %q", got)
	}
	if got := i.Request.Headers.Get("User-Agent"); got != "MgcTF/test" {
		t.Errorf("User-Agent should be preserved, got %q", got)
	}
	if got := i.Response.Headers.Get("Date"); got != "" {
		t.Errorf("volatile Date header not dropped: %q", got)
	}
	if got := i.Response.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type should be preserved, got %q", got)
	}
	for _, secret := range []string{"super-secret", "also-secret"} {
		if strings.Contains(i.Request.Body, secret) {
			t.Errorf("request body still contains secret %q: %s", secret, i.Request.Body)
		}
	}
	if strings.Contains(i.Response.Body, "leaky") {
		t.Errorf("response body still contains token: %s", i.Response.Body)
	}
	// Non-secret fields survive.
	if !strings.Contains(i.Request.Body, `"name":"x"`) {
		t.Errorf("request body lost a non-secret field: %s", i.Request.Body)
	}
}

func TestScrubHookKubeconfigAndPEM(t *testing.T) {
	t.Parallel()

	// A kubeconfig response body (YAML, not JSON) carries cluster-admin
	// credentials; every data field and token must be redacted, structure kept.
	kubeconfig := `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNhVENDQWRJPQ==
    server: https://cluster.example.com:6443
  name: test
users:
- name: admin
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNiVENDQWRJPQ==
    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBPQ==
    token: super-secret-bearer
`
	i := &cassette.Interaction{
		Request:  cassette.Request{Headers: http.Header{}},
		Response: cassette.Response{Headers: http.Header{}, Body: kubeconfig},
	}
	if err := scrubHook(i); err != nil {
		t.Fatalf("scrubHook: %v", err)
	}
	for _, leak := range []string{"LS0tLS1", "super-secret-bearer"} {
		if strings.Contains(i.Response.Body, leak) {
			t.Errorf("kubeconfig credential %q survived the scrub: %s", leak, i.Response.Body)
		}
	}
	if !strings.Contains(i.Response.Body, "server: https://cluster.example.com:6443") {
		t.Errorf("scrub destroyed non-secret kubeconfig content: %s", i.Response.Body)
	}

	// Raw PEM blocks anywhere in a body are redacted wholesale.
	pem := "before\n-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA\n-----END RSA PRIVATE KEY-----\nafter"
	j := &cassette.Interaction{
		Request:  cassette.Request{Headers: http.Header{}, Body: pem},
		Response: cassette.Response{Headers: http.Header{}},
	}
	if err := scrubHook(j); err != nil {
		t.Fatalf("scrubHook: %v", err)
	}
	if strings.Contains(j.Request.Body, "BEGIN RSA") || strings.Contains(j.Request.Body, "MIIEpAIB") {
		t.Errorf("PEM block survived the scrub: %s", j.Request.Body)
	}
	if !strings.Contains(j.Request.Body, "before") || !strings.Contains(j.Request.Body, "after") {
		t.Errorf("scrub destroyed content around the PEM block: %s", j.Request.Body)
	}

	// A kubeconfig embedded as a JSON string field is redacted by key.
	k := &cassette.Interaction{
		Request:  cassette.Request{Headers: http.Header{}},
		Response: cassette.Response{Headers: http.Header{}, Body: `{"kubeconfig":"apiVersion: v1\nusers: ...","name":"x"}`},
	}
	if err := scrubHook(k); err != nil {
		t.Fatalf("scrubHook: %v", err)
	}
	if strings.Contains(k.Response.Body, "apiVersion") {
		t.Errorf("JSON kubeconfig field survived the scrub: %s", k.Response.Body)
	}
	if !strings.Contains(k.Response.Body, `"name":"x"`) {
		t.Errorf("scrub destroyed a non-secret JSON field: %s", k.Response.Body)
	}
}

func TestGuardHook(t *testing.T) {
	t.Parallel()

	clean := &cassette.Interaction{
		Request:  cassette.Request{Method: "POST", URL: "https://h/v1/clusters", Headers: http.Header{}, Body: `{"name":"tf-acctest-x"}`},
		Response: cassette.Response{Headers: http.Header{}, Body: `{"id":"abc","status":"provisioning"}`},
	}
	if err := guardHook(clean); err != nil {
		t.Errorf("clean interaction flagged as leak: %v", err)
	}

	leaks := []struct {
		name string
		body string
	}{
		{name: "pem block", body: "-----BEGIN CERTIFICATE-----\nMIIC"},
		{name: "jwt", body: `{"weird_field":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"}`},
		{name: "long base64 blob", body: `{"ca_bundle":"` + strings.Repeat("A1b2C3d4", 40) + `"}`},
		{name: "unscrubbed key data", body: `{"nested":{"client-key-data":"TUlJRXBBSUJBQUtDQVFFQQ=="}}`},
	}
	for _, tt := range leaks {
		i := &cassette.Interaction{
			Request:  cassette.Request{Method: "GET", URL: "https://h/v1/x", Headers: http.Header{}},
			Response: cassette.Response{Headers: http.Header{}, Body: tt.body},
		}
		if err := guardHook(i); err == nil {
			t.Errorf("%s: credential-like content not flagged", tt.name)
		}
	}

	// The guard also inspects surviving header values.
	h := &cassette.Interaction{
		Request:  cassette.Request{Method: "GET", URL: "https://h/v1/x", Headers: http.Header{}},
		Response: cassette.Response{Headers: http.Header{}, Body: "{}"},
	}
	h.Response.Headers.Set("X-Custom-Auth", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIn0.dozjgNryP4J3jVmNHl0w5N")
	if err := guardHook(h); err == nil {
		t.Error("JWT in a surviving header not flagged")
	}
}

func TestApplyVCR(t *testing.T) {
	t.Parallel()
	client := &http.Client{Transport: http.DefaultTransport}

	// Replay: the recorder client is injected, retry backoff is flattened and
	// polling is forced fast (interactions advance per request, not per time).
	replay := applyVCR(utils.DataConfig{CoreConfig: *sdk.NewMgcClient(sdk.WithAPIKey("x"))}, client, false)
	if got := replay.CoreConfig.GetConfig().HTTPClient; got != client {
		t.Error("replay: recorder client was not injected")
	}
	if iv := replay.CoreConfig.GetConfig().RetryConfig.InitialInterval; iv != 0 {
		t.Errorf("replay: retry interval not flattened, got %v", iv)
	}
	if replay.PollingInterval != replayPollingInterval {
		t.Errorf("replay: polling interval not forced, got %v", replay.PollingInterval)
	}
	if replay.PollingTimeout != replayPollingTimeout {
		t.Errorf("replay: polling timeout not forced, got %v", replay.PollingTimeout)
	}

	// Record: the client is injected but retry and polling keep the real-world
	// pace (faithful cassettes; the profile env decides the polling).
	record := applyVCR(utils.DataConfig{CoreConfig: *sdk.NewMgcClient(sdk.WithAPIKey("x"))}, client, true)
	if got := record.CoreConfig.GetConfig().HTTPClient; got != client {
		t.Error("record: recorder client was not injected")
	}
	if iv := record.CoreConfig.GetConfig().RetryConfig.InitialInterval; iv != sdk.DefaultInitialInterval {
		t.Errorf("record: retry interval should keep SDK default, got %v", iv)
	}
	if record.PollingInterval != 0 || record.PollingTimeout != 0 {
		t.Errorf("record: polling must stay unset (env/profile decides), got %v/%v",
			record.PollingInterval, record.PollingTimeout)
	}

	// A CoreFor derivation (custom endpoint) shares the same recorder client, so
	// there is one recorder per test across every service.
	replay.SetServiceEndpoints(map[string]string{utils.ServiceKubernetes: "https://k8s.example.com"})
	if got := replay.CoreFor(utils.ServiceKubernetes).GetConfig().HTTPClient; got != client {
		t.Error("CoreFor derivation should share the recorder client")
	}
}

func TestCassettePath(t *testing.T) {
	base := t.TempDir()
	t.Setenv(EnvVCRPath, base)

	// Staging is the only local cassette set: record, live and replay all
	// resolve to it, namespaced by the package dir go test runs in ("acctest").
	want := filepath.Join(base, StagingDir, "acctest", "TestCassettePath")
	if got := cassettePath(t); got != want {
		t.Errorf("cassette path = %q, want %q", got, want)
	}
}

func TestRandomNameDeterministicAcrossRuns(t *testing.T) {
	t.Parallel()
	// Two harnesses seeded from the same test name must produce the same names,
	// which is what keeps a recording and its replay byte-for-byte identical.
	const name = "TestAccSomething_basic"
	a := &VCR{rnd: rand.New(rand.NewPCG(fnvSeed(name), fnvSeed(name)))}
	b := &VCR{rnd: rand.New(rand.NewPCG(fnvSeed(name), fnvSeed(name)))}

	for i := 0; i < 3; i++ {
		na, nb := a.RandomName("tfacc"), b.RandomName("tfacc")
		if na != nb {
			t.Fatalf("call %d: names diverged: %q vs %q", i, na, nb)
		}
		wantPrefix := ResourcePrefix + "-tfacc-"
		if !strings.HasPrefix(na, wantPrefix) || len(na) != len(wantPrefix)+8 {
			t.Fatalf("unexpected name shape: %q", na)
		}
	}

	// A different test name must yield a different sequence.
	c := &VCR{rnd: rand.New(rand.NewPCG(fnvSeed("Other"), fnvSeed("Other")))}
	if a2, c1 := (&VCR{rnd: rand.New(rand.NewPCG(fnvSeed(name), fnvSeed(name)))}).RandomName("p"), c.RandomName("p"); a2 == c1 {
		t.Fatalf("different test names produced identical names: %q", a2)
	}
}
