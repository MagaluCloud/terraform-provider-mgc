package acctest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	sdk "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

const (
	EnvVCRMode  = "MGC_VCR_MODE"
	EnvVCRPath  = "MGC_VCR_PATH"
	cassetteExt = ".yaml"
)

const providerName = "mgc"

const (
	replayPollingInterval = 10 * time.Millisecond
	replayPollingTimeout  = 5 * time.Minute
)

type VCR struct {
	rec    *recorder.Recorder
	client *http.Client
	rnd    *rand.Rand
}

func NewVCR(t *testing.T) *VCR {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test skipped: set TF_ACC to run")
	}
	if testing.Short() {
		t.Skip("skipping acceptance test in -short mode")
	}

	mode := vcrMode(t)
	if mode != recorder.ModeReplayOnly && Endpoint() == "" {
		t.Fatalf("%s must be set when not replaying: acceptance tests no longer fall back to the provider's default (production) endpoint — point %s at the target API before recording or running live",
			EnvEndpoint, EnvEndpoint)
	}
	if mode == recorder.ModeRecordOnly && isLoopbackEndpoint(Endpoint()) {
		t.Fatalf("refusing to record against fake endpoint %q: published cassettes must come from the real API so replay stays faithful — set %s to a real API host",
			Endpoint(), EnvEndpoint)
	}
	cassetteName := cassettePath(t, mode)
	if mode != recorder.ModeReplayOnly {
		if err := os.MkdirAll(filepath.Dir(cassetteName), 0o755); err != nil {
			t.Fatalf("creating cassette dir: %v", err)
		}
	}

	rec, err := recorder.New(cassetteName,
		recorder.WithMode(mode),
		recorder.WithRealTransport(http.DefaultTransport),
		recorder.WithSkipRequestLatency(true),
		recorder.WithMatcher(matcher),
		recorder.WithHook(scrubHook, recorder.BeforeSaveHook),
		recorder.WithHook(guardHook, recorder.BeforeSaveHook),
	)
	if err != nil {
		if mode == recorder.ModeReplayOnly {
			t.Fatalf("loading cassette %s.yaml: %v\nfetch the published set with `make cassettes-download`, or record it with `make testacc-record RUN=%s`",
				cassetteName, err, t.Name())
		}
		t.Fatalf("creating VCR recorder: %v", err)
	}

	t.Cleanup(func() {
		recording := rec.IsRecording()
		if recording && t.Failed() {
			t.Logf("test failed while recording: %s", cassetteName+cassetteExt)
			return
		}
		if err := rec.Stop(); err != nil {
			t.Errorf("stopping VCR recorder: %v", err)
		}
		if recording {
			discardCassetteIfEmpty(t, cassetteName)
		}
	})

	seed := fnvSeed(t.Name())
	return &VCR{
		rec:    rec,
		client: &http.Client{Transport: rec},
		rnd:    rand.New(rand.NewPCG(seed, seed)),
	}
}

func parseVCRMode(raw string) (recorder.Mode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "replay":
		return recorder.ModeReplayOnly, nil
	case "record":
		return recorder.ModeRecordOnly, nil
	case "off", "live", "passthrough":
		return recorder.ModePassthrough, nil
	default:
		return 0, fmt.Errorf("invalid %s=%q (want replay|record|off)", EnvVCRMode, raw)
	}
}

func vcrMode(t *testing.T) recorder.Mode {
	m, err := parseVCRMode(os.Getenv(EnvVCRMode))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func isReplay() bool {
	m, err := parseVCRMode(os.Getenv(EnvVCRMode))
	return err == nil && m == recorder.ModeReplayOnly
}

// cassettePath resolves where this test's cassette lives (without the .yaml
// extension the recorder appends), laid out as <base>/<set>/<service>/<TestName>,
// service being the package dir go test runs in.
func cassettePath(t *testing.T, mode recorder.Mode) string {
	base, err := VCRBase()
	if err != nil {
		t.Fatal(err)
	}
	var set string
	if mode != recorder.ModePassthrough {
		if set, err = CurrentSet(); err != nil {
			t.Fatal(err)
		}
	}
	path, err := resolveCassette(base, set, filepath.Join(serviceDir(t), sanitizeName(t.Name())), mode)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func resolveCassette(base, set, rel string, mode recorder.Mode) (string, error) {
	mainPath := filepath.Join(base, MainSet, rel)
	if mode == recorder.ModePassthrough {
		return mainPath, nil
	}

	branchPath := filepath.Join(base, set, rel)
	if mode == recorder.ModeRecordOnly {
		if set == MainSet {
			return "", fmt.Errorf("refusing to record with %q checked out: the published set is only written by promoting a merged branch — record from a branch", MainSet)
		}
		return branchPath, nil
	}
	if _, err := os.Stat(branchPath + cassetteExt); err == nil {
		return branchPath, nil
	}
	return mainPath, nil
}

func discardCassetteIfEmpty(t *testing.T, name string) {
	c, err := cassette.Load(name)
	if err != nil {
		return
	}
	if len(c.Interactions) == 0 {
		if err := os.Remove(c.File); err != nil {
			t.Logf("removing empty cassette %s: %v", c.File, err)
		}
	}
}

func serviceDir(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolving test working directory: %v", err)
	}
	return filepath.Base(wd)
}

func VCRBase() (string, error) {
	if p := os.Getenv(EnvVCRPath); p != "" {
		return p, nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolving user cache dir for cassettes (set %s to override): %w", EnvVCRPath, err)
	}
	return filepath.Join(cache, "terraform-provider-mgc", "cassettes"), nil
}

func isLoopbackEndpoint(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func SkipIfVcr(t *testing.T) {
	t.Helper()
	if vcrMode(t) != recorder.ModePassthrough {
		t.Skipf("test is not VCR-compatible; run it live with %s=off", EnvVCRMode)
	}
}

func (v *VCR) HTTPClient() *http.Client { return v.client }

func (v *VCR) ProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		providerName: providerserver.NewProtocol6WithError(&vcrProvider{
			inner: mgc.New("test")(),
			vcr:   v,
		}),
	}
}

type vcrProvider struct {
	inner provider.Provider
	vcr   *VCR
}

func (p *vcrProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	p.inner.Metadata(ctx, req, resp)
}

func (p *vcrProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	p.inner.Schema(ctx, req, resp)
}

func (p *vcrProvider) Resources(ctx context.Context) []func() resource.Resource {
	return p.inner.Resources(ctx)
}

func (p *vcrProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return p.inner.DataSources(ctx)
}

func (p *vcrProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	p.inner.Configure(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, ok := resp.ResourceData.(utils.DataConfig)
	if !ok {
		return
	}
	cfg = applyVCR(cfg, p.vcr.client, p.vcr.rec.IsRecording())
	resp.ResourceData = cfg
	resp.DataSourceData = cfg
}

func applyVCR(cfg utils.DataConfig, client *http.Client, recording bool) utils.DataConfig {
	conf := cfg.CoreConfig.GetConfig()
	conf.HTTPClient = client
	if !recording {
		conf.RetryConfig = sdk.RetryConfig{
			MaxAttempts:     sdk.DefaultMaxAttempts,
			InitialInterval: 0,
			MaxInterval:     0,
			BackoffFactor:   1,
		}
		cfg.PollingInterval = replayPollingInterval
		cfg.PollingTimeout = replayPollingTimeout
	}
	return cfg
}

// PollInterval and PollTimeout give out-of-band SDK poll loops in tests (the
// disappears/destroy checks, which bypass the provider and so never see
// applyVCR) the same replay behavior the provider gets: the fast replay pace
// when replaying a cassette — interactions advance per request, not per
// wall-clock — and def otherwise (record/live poll for real).
func PollInterval(def time.Duration) time.Duration {
	if isReplay() {
		return replayPollingInterval
	}
	return def
}

func PollTimeout(def time.Duration) time.Duration {
	if isReplay() {
		return replayPollingTimeout
	}
	return def
}

// replayPlaceholderURL is the base URL for out-of-band SDK clients during
// replay when no endpoint is configured. The .invalid TLD is reserved and never
// resolves (RFC 6761), and the recorder serves every response anyway — nothing
// leaves the process. Record/live without an endpoint is refused in NewVCR.
const replayPlaceholderURL = "https://acctest.invalid"

// SDKClient returns a CoreClient wired to this test's recorder transport, so
// out-of-band SDK checks (exists/destroy/disappears) are recorded and replayed
// together with the provider traffic.
func (v *VCR) SDKClient(service string) *sdk.CoreClient {
	url := EndpointFor(service)
	if url == "" {
		url = replayPlaceholderURL
	}
	return sdk.NewMgcClient(
		sdk.WithAPIKey(effectiveAPIKey()),
		sdk.WithBaseURL(sdk.MgcUrl(url)),
		sdk.WithHTTPClient(v.client),
	)
}

func (v *VCR) RandomName(kind string) string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = nameAlphabet[v.rnd.IntN(len(nameAlphabet))]
	}
	return ResourcePrefix + "-" + kind + "-" + string(b)
}

func fnvSeed(name string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return h.Sum64()
}

var nonNameRe = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

func sanitizeName(name string) string {
	return nonNameRe.ReplaceAllString(name, "_")
}

// regionSegmentRe drops a leading region path segment (e.g. /br-se1/...):
// prod URLs carry the region in the path and a cassette must replay against any of them.
var regionSegmentRe = regexp.MustCompile(`^/br-[a-z0-9-]+/`)

func matcher(r *http.Request, i cassette.Request) bool {
	if r.Method != i.Method {
		return false
	}

	recURL, err := url.Parse(i.URL)
	if err != nil {
		return false
	}
	if normalizePath(r.URL.Path) != normalizePath(recURL.Path) {
		return false
	}
	if sortedQuery(r.URL.RawQuery) != sortedQuery(recURL.RawQuery) {
		return false
	}

	return bodyEqual(readAndRestoreBody(r), i.Body)
}

func normalizePath(p string) string {
	p = strings.TrimSuffix(p, "/")
	p = regionSegmentRe.ReplaceAllString(p, "/")
	return p
}

func sortedQuery(raw string) string {
	vals, err := url.ParseQuery(raw)
	if err != nil {
		return raw
	}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sort.Strings(vals[k])
		for _, val := range vals[k] {
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(val)
			sb.WriteByte('&')
		}
	}
	return sb.String()
}

func readAndRestoreBody(r *http.Request) []byte {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil
	}
	r.Body = io.NopCloser(bytes.NewReader(b))
	return b
}

func bodyEqual(live []byte, recorded string) bool {
	if len(bytes.TrimSpace(live)) == 0 && strings.TrimSpace(recorded) == "" {
		return true
	}
	var a, b any
	if json.Unmarshal(live, &a) == nil && json.Unmarshal([]byte(recorded), &b) == nil {
		return reflect.DeepEqual(a, b)
	}
	return string(live) == recorded
}

var (
	secretHeaders       = []string{"Authorization", "X-Api-Key", "X-API-Key"}
	volatileRespHeaders = []string{"Date", "X-Request-Id", "X-Mgc-Trace-Id", "Set-Cookie"}
	secretBodyKeys      = map[string]struct{}{
		"api_key":                    {},
		"key_pair_secret":            {},
		"token":                      {},
		"password":                   {},
		"kubeconfig":                 {},
		"private_key":                {},
		"client-key-data":            {},
		"client_key_data":            {},
		"client-certificate-data":    {},
		"client_certificate_data":    {},
		"certificate-authority-data": {},
		"certificate_authority_data": {},
	}

	pemBlockRe   = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]+-----[\s\S]*?-----END [A-Z0-9 ]+-----`)
	textSecretRe = regexp.MustCompile(`(?im)^(\s*(?:client-key-data|client-certificate-data|certificate-authority-data|token|password)\s*:\s*)\S+`)
)

func scrubHook(i *cassette.Interaction) error {
	for _, h := range secretHeaders {
		i.Request.Headers.Del(h)
	}
	for _, h := range volatileRespHeaders {
		i.Response.Headers.Del(h)
	}
	i.Request.Body = redactBody(i.Request.Body)
	i.Response.Body = redactBody(i.Response.Body)
	return nil
}

func redactBody(body string) string {
	body = redactJSON(body)
	body = pemBlockRe.ReplaceAllString(body, "REDACTED")
	body = textSecretRe.ReplaceAllString(body, "${1}REDACTED")
	return body
}

var credentialLeakRes = []*regexp.Regexp{
	regexp.MustCompile(`-----BEGIN `),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
	regexp.MustCompile(`(?i)(?:client|certificate)[-_](?:key|certificate|authority)[-_]data\W{0,3}[A-Za-z0-9+/=]{16,}`),
	regexp.MustCompile(`[A-Za-z0-9+/]{256,}`),
}

func findCredential(s string) string {
	for _, re := range credentialLeakRes {
		if m := re.FindString(s); m != "" {
			if len(m) > 40 {
				m = m[:40] + "..."
			}
			return m
		}
	}
	return ""
}

// guardHook runs after scrubHook and blocks the cassette save when anything
// credential-shaped survived: cassettes leave the machine, so an unknown
// secret must fail loudly instead of leaking silently.
func guardHook(i *cassette.Interaction) error {
	leak := func(where, s string) error {
		if m := findCredential(s); m != "" {
			return fmt.Errorf("credential-like content in %s of %s %s (matched %q): extend scrubHook before this cassette can be saved",
				where, i.Request.Method, i.Request.URL, m)
		}
		return nil
	}

	if err := leak("request body", i.Request.Body); err != nil {
		return err
	}
	if err := leak("response body", i.Response.Body); err != nil {
		return err
	}
	for _, headers := range []http.Header{i.Request.Headers, i.Response.Headers} {
		for name, values := range headers {
			for _, v := range values {
				if err := leak("header "+name, v); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ScanCassetteFile reports credential-like content in a saved cassette so the
// publish tooling can refuse to upload it.
func ScanCassetteFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if m := findCredential(string(data)); m != "" {
		return fmt.Errorf("%s: credential-like content (matched %q)", path, m)
	}
	return nil
}

func redactJSON(body string) string {
	if strings.TrimSpace(body) == "" {
		return body
	}
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return body
	}
	if !redactWalk(v) {
		return body
	}
	out, err := json.Marshal(v)
	if err != nil {
		return body
	}
	return string(out)
}

func redactWalk(v any) bool {
	changed := false
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if _, secret := secretBodyKeys[strings.ToLower(k)]; secret {
				if _, isStr := val.(string); isStr {
					t[k] = "REDACTED"
					changed = true
					continue
				}
			}
			if redactWalk(val) {
				changed = true
			}
		}
	case []any:
		for _, val := range t {
			if redactWalk(val) {
				changed = true
			}
		}
	}
	return changed
}
