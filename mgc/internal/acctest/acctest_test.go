package acctest

import (
	"strings"
	"testing"
)

func TestProviderConfigPolling(t *testing.T) {
	// ProviderConfig reads region/api_key from the env; pin them so the test
	// exercises only the polling injection.
	t.Setenv(EnvRegion, "br-se1")
	t.Setenv(EnvAPIKey, "00000000-0000-4000-8000-000000000000")

	t.Run("omitted when env is unset", func(t *testing.T) {
		t.Setenv(EnvEndpoint, "")
		t.Setenv(EnvPollingInterval, "")
		t.Setenv(EnvPollingTimeout, "")

		cfg := ProviderConfig()
		if strings.Contains(cfg, "polling_interval") || strings.Contains(cfg, "polling_timeout") {
			t.Errorf("expected no polling attributes, got:\n%s", cfg)
		}
	})

	t.Run("emitted from env (no endpoints branch)", func(t *testing.T) {
		t.Setenv(EnvEndpoint, "")
		t.Setenv(EnvPollingInterval, "200ms")
		t.Setenv(EnvPollingTimeout, "5m")

		cfg := ProviderConfig()
		if !strings.Contains(cfg, `polling_interval = "200ms"`) {
			t.Errorf("missing polling_interval:\n%s", cfg)
		}
		if !strings.Contains(cfg, `polling_timeout = "5m"`) {
			t.Errorf("missing polling_timeout:\n%s", cfg)
		}
	})

	t.Run("emitted from env (endpoints branch)", func(t *testing.T) {
		t.Setenv(EnvEndpoint, "http://localhost:8080")
		t.Setenv(EnvPollingInterval, "200ms")
		t.Setenv(EnvPollingTimeout, "")

		cfg := ProviderConfig("kubernetes")
		if !strings.Contains(cfg, `polling_interval = "200ms"`) {
			t.Errorf("missing polling_interval in endpoints branch:\n%s", cfg)
		}
		if strings.Contains(cfg, "polling_timeout") {
			t.Errorf("unset timeout should be omitted:\n%s", cfg)
		}
	})
}
