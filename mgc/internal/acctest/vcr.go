package acctest

import (
	"bytes"
	"context"
	"encoding/json"
	"hash/fnv"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

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

const EnvVCRMode = "MGC_VCR_MODE"

const providerName = "mgc"

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

	cassetteName := filepath.Join("testdata", "cassettes", sanitizeName(t.Name()))

	rec, err := recorder.New(cassetteName,
		recorder.WithMode(vcrMode(t)),
		recorder.WithRealTransport(http.DefaultTransport),
		recorder.WithSkipRequestLatency(true),
		recorder.WithMatcher(matcher),
		recorder.WithHook(scrubHook, recorder.BeforeSaveHook),
	)
	if err != nil {
		t.Fatalf("creating VCR recorder: %v", err)
	}

	t.Cleanup(func() {
		if err := rec.Stop(); err != nil {
			t.Errorf("stopping VCR recorder: %v", err)
		}
	})

	seed := fnvSeed(t.Name())
	return &VCR{
		rec:    rec,
		client: &http.Client{Transport: rec},
		rnd:    rand.New(rand.NewPCG(seed, seed)),
	}
}

func vcrMode(t *testing.T) recorder.Mode {
	raw := os.Getenv(EnvVCRMode)
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "auto":
		return recorder.ModeRecordOnce
	case "record":
		return recorder.ModeRecordOnly
	case "replay":
		return recorder.ModeReplayOnly
	case "off", "live", "passthrough":
		return recorder.ModePassthrough
	default:
		t.Fatalf("invalid %s=%q (want auto|record|replay|off)", EnvVCRMode, raw)
		return recorder.ModeRecordOnce
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
	}
	return cfg
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

var uuidRe = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

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
	return uuidRe.ReplaceAllString(p, "{id}")
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
		"api_key":         {},
		"key_pair_secret": {},
		"token":           {},
		"password":        {},
	}
)

func scrubHook(i *cassette.Interaction) error {
	for _, h := range secretHeaders {
		i.Request.Headers.Del(h)
	}
	for _, h := range volatileRespHeaders {
		i.Response.Headers.Del(h)
	}
	i.Request.Body = redactJSON(i.Request.Body)
	i.Response.Body = redactJSON(i.Response.Body)
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
