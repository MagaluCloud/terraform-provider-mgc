package acctest

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

const (
	EnvEndpoint    = "MGC_ENDPOINT"
	EnvAPIKey      = "MGC_API_KEY"
	EnvRegion      = "MGC_REGION"
	defaultRegion  = "br-se1"
	ResourcePrefix = "tf-acctest"
	replayAPIKey   = "00000000-0000-4000-8000-000000000000"

	// EnvPollingInterval and EnvPollingTimeout let a profile speed up recording
	// against the fake by overriding the resources' slow real-world polling
	// defaults through the provider config. Replay ignores them (applyVCR forces
	// its own pace); a real provider user never sets them.
	EnvPollingInterval = "MGC_POLLING_INTERVAL"
	EnvPollingTimeout  = "MGC_POLLING_TIMEOUT"
)

var globalServices = map[string]bool{
	utils.ServiceSSH: true,
}

func ProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"mgc": providerserver.NewProtocol6WithError(mgc.New("test")()),
	}
}

func PreCheck(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping acceptance test in -short mode")
	}

	if !isReplay() && APIKey() == "" {
		t.Fatalf("%s must be set (any valid UUIDv4 for a fake API, a real key for production)", EnvAPIKey)
	}

	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" && os.Getenv("TF_ACC_TERRAFORM_VERSION") == "" {
		if _, err := exec.LookPath("terraform"); err != nil {
			t.Skip("terraform binary not found in PATH; install it or set TF_ACC_TERRAFORM_PATH")
		}
	}
}

func Endpoint() string {
	return strings.TrimSuffix(os.Getenv(EnvEndpoint), "/")
}

func EndpointFor(service string) string {
	base := Endpoint()
	if base == "" || globalServices[service] {
		return base
	}
	return base + "/" + Region()
}

func APIKey() string {
	return os.Getenv(EnvAPIKey)
}

func effectiveAPIKey() string {
	if key := APIKey(); key != "" {
		return key
	}
	if isReplay() {
		return replayAPIKey
	}
	return ""
}

func Region() string {
	if region := os.Getenv(EnvRegion); region != "" {
		return region
	}
	return defaultRegion
}

func ProviderConfig(services ...string) string {
	polling := pollingConfig()
	if Endpoint() == "" {
		return fmt.Sprintf(`
provider "mgc" {
  api_key = %q
  region  = %q
%s}
`, effectiveAPIKey(), Region(), polling)
	}

	var endpoints strings.Builder
	for _, service := range services {
		fmt.Fprintf(&endpoints, "    %s = %q\n", service, EndpointFor(service))
	}
	return fmt.Sprintf(`
provider "mgc" {
  api_key = %q
  region  = %q
%s  endpoints {
%s  }
}
`, effectiveAPIKey(), Region(), polling, endpoints.String())
}

// pollingConfig emits provider polling attributes from the environment so a
// profile can override the resources' slow real-world defaults when recording
// against the fake. Empty when unset — replay stays hermetic and applyVCR
// forces its own fast pace regardless.
func pollingConfig() string {
	var b strings.Builder
	if v := os.Getenv(EnvPollingInterval); v != "" {
		fmt.Fprintf(&b, "  polling_interval = %q\n", v)
	}
	if v := os.Getenv(EnvPollingTimeout); v != "" {
		fmt.Fprintf(&b, "  polling_timeout = %q\n", v)
	}
	return b.String()
}

const nameAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func RandomName(kind string) string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = nameAlphabet[rand.IntN(len(nameAlphabet))]
	}
	return ResourcePrefix + "-" + kind + "-" + string(b)
}

func IsSweepableTestResource(name string) bool {
	return strings.HasPrefix(name, ResourcePrefix+"-")
}
