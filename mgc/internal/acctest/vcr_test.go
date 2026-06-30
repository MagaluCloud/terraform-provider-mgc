package acctest

import (
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
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
			name: "uuid in path is normalized",
			req:  newReq(t, http.MethodGet, "https://h/v1/clusters/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ""),
			rec:  cassette.Request{Method: "GET", URL: "https://h/v1/clusters/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Body: ""},
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

func TestApplyVCR(t *testing.T) {
	t.Parallel()
	client := &http.Client{Transport: http.DefaultTransport}

	// Replay: the recorder client is injected and retry backoff is flattened.
	replay := applyVCR(utils.DataConfig{CoreConfig: *sdk.NewMgcClient(sdk.WithAPIKey("x"))}, client, false)
	if got := replay.CoreConfig.GetConfig().HTTPClient; got != client {
		t.Error("replay: recorder client was not injected")
	}
	if iv := replay.CoreConfig.GetConfig().RetryConfig.InitialInterval; iv != 0 {
		t.Errorf("replay: retry interval not flattened, got %v", iv)
	}

	// Record: the client is injected but retry keeps the SDK defaults (faithful).
	record := applyVCR(utils.DataConfig{CoreConfig: *sdk.NewMgcClient(sdk.WithAPIKey("x"))}, client, true)
	if got := record.CoreConfig.GetConfig().HTTPClient; got != client {
		t.Error("record: recorder client was not injected")
	}
	if iv := record.CoreConfig.GetConfig().RetryConfig.InitialInterval; iv != sdk.DefaultInitialInterval {
		t.Errorf("record: retry interval should keep SDK default, got %v", iv)
	}

	// A CoreFor derivation (custom endpoint) shares the same recorder client, so
	// there is one recorder per test across every service.
	replay.SetServiceEndpoints(map[string]string{utils.ServiceKubernetes: "https://k8s.example.com"})
	if got := replay.CoreFor(utils.ServiceKubernetes).GetConfig().HTTPClient; got != client {
		t.Error("CoreFor derivation should share the recorder client")
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
